package yapi

import (
	"bytes"
	"html"
	"regexp"
	"strings"

	model "github.com/tangcent/apilot/api-model"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldmarkhtml "github.com/yuin/goldmark/renderer/html"
)

// Interface is one YApi interface, in the shape YApi's /api/interface/save
// takes it. Token, CatID and ID are filled in when it is saved.
type Interface struct {
	ID                  int64       `json:"id,omitempty"`
	Token               string      `json:"token,omitempty"`
	CatID               int64       `json:"catid,omitempty"`
	Title               string      `json:"title"`
	Path                string      `json:"path"`
	Method              string      `json:"method"`
	Desc                string      `json:"desc"`
	Markdown            string      `json:"markdown"`
	Status              string      `json:"status"`
	ReqBodyIsJSONSchema bool        `json:"req_body_is_json_schema"`
	ResBodyIsJSONSchema bool        `json:"res_body_is_json_schema"`
	ReqHeaders          []Header    `json:"req_headers"`
	ReqQuery            []Query     `json:"req_query"`
	ReqParams           []PathParam `json:"req_params"`
	ReqBodyForm         []FormField `json:"req_body_form"`
	ReqBodyType         string      `json:"req_body_type,omitempty"`
	ReqBodyOther        string      `json:"req_body_other"`
	ResBodyType         string      `json:"res_body_type,omitempty"`
	ResBody             string      `json:"res_body"`

	// category is the YApi category the interface is saved in.
	category string
}

// Header is a request header. Required is "1" or "0", as YApi stores it.
type Header struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Desc     string `json:"desc"`
	Example  string `json:"example"`
	Required string `json:"required"`
}

// Query is a query-string parameter.
type Query struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Example  string `json:"example"`
	Desc     string `json:"desc"`
	Required string `json:"required"`
}

// PathParam is a path variable.
type PathParam struct {
	Name    string `json:"name"`
	Example string `json:"example"`
	Desc    string `json:"desc"`
}

// FormField is a form or multipart field; Type is "text" or "file".
type FormField struct {
	Name     string `json:"name"`
	Example  string `json:"example"`
	Type     string `json:"type"`
	Required string `json:"required"`
	Desc     string `json:"desc"`
}

// Category returns the YApi category the interface belongs to.
func (it Interface) Category() string { return it.category }

// buildInterface converts one endpoint.
//
// The title is the first line of the endpoint's description and the rest is
// the interface description; the category is the first line of the class's
// doc comment, or the class name. This is how the EasyYapi IntelliJ plugin
// names them, so an export from either tool lands in the same place.
func buildInterface(ep model.ApiEndpoint, b *schemaBuilder) Interface {
	title, rest := splitDoc(ep.Description)
	if title == "" {
		title = ep.Name
	}
	method := strings.ToUpper(ep.Method)
	if method == "" {
		method = "GET"
	}
	if title == "" {
		title = method + " " + ep.Path
	}
	it := Interface{
		Title:               title,
		Path:                normalizePath(ep.Path),
		Method:              strings.ToLower(method),
		Markdown:            rest,
		Desc:                renderHTML(rest),
		Status:              "done",
		ReqBodyIsJSONSchema: true,
		ResBodyIsJSONSchema: true,
		ReqHeaders:          []Header{},
		ReqQuery:            []Query{},
		ReqParams:           []PathParam{},
		ReqBodyForm:         []FormField{},
		category:            categoryOf(ep),
	}

	hasFile := false
	for _, p := range ep.Parameters {
		switch p.In {
		case "query":
			it.ReqQuery = append(it.ReqQuery, Query{
				Name: p.Name, Value: p.Default, Example: firstNonEmpty(p.Example, p.Default, parameterMock(p.Name)),
				Desc: p.Description, Required: flag(p.Required),
			})
		case "path":
			it.ReqParams = append(it.ReqParams, PathParam{Name: p.Name, Example: firstNonEmpty(p.Example, parameterMock(p.Name)), Desc: p.Description})
		case "form":
			kind := "text"
			if p.Type == "file" {
				kind = "file"
				hasFile = true
			}
			it.ReqBodyForm = append(it.ReqBodyForm, FormField{
				Name: p.Name, Example: p.Example, Type: kind, Required: flag(p.Required), Desc: p.Description,
			})
		case "header":
			it.ReqHeaders = append(it.ReqHeaders, Header{
				Name: p.Name, Value: p.Default, Desc: p.Description, Example: firstNonEmpty(p.Example, p.Default), Required: flag(p.Required),
			})
		}
	}
	for _, h := range ep.Headers {
		it.ReqHeaders = append(it.ReqHeaders, Header{
			Name: h.Name, Value: h.Value, Desc: h.Description, Example: firstNonEmpty(h.Example, h.Value), Required: flag(h.Required),
		})
	}

	switch {
	case ep.RequestBody != nil:
		it.ReqBodyType = "json"
		it.ReqBodyOther = b.documentString(ep.RequestBody.Body)
		addContentType(&it, firstNonEmpty(first(ep.Consumes), "application/json"))
	case len(it.ReqBodyForm) > 0:
		it.ReqBodyType = "form"
		contentType := "application/x-www-form-urlencoded"
		if hasFile {
			contentType = "multipart/form-data"
		}
		addContentType(&it, firstNonEmpty(first(ep.Consumes), contentType))
	}
	if ep.Response != nil {
		it.ResBodyType = "json"
		it.ResBody = b.documentString(ep.Response.Body)
	}
	return it
}

func addContentType(it *Interface, contentType string) {
	for _, h := range it.ReqHeaders {
		if strings.EqualFold(h.Name, "Content-Type") {
			return
		}
	}
	it.ReqHeaders = append(it.ReqHeaders, Header{
		Name: "Content-Type", Value: contentType, Example: contentType, Required: "0",
	})
}

func categoryOf(ep model.ApiEndpoint) string {
	if title, _ := splitDoc(ep.FolderDescription); title != "" {
		return title
	}
	if ep.Folder != "" {
		return ep.Folder
	}
	return "anonymous"
}

// splitDoc splits a description into its first line and the rest.
func splitDoc(doc string) (string, string) {
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(doc, "\r\n", "\n"), "\n") {
		lines = append(lines, strings.TrimRight(line, " \t"))
	}
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		return strings.TrimSpace(line), strings.TrimSpace(strings.Join(lines[i+1:], "\n"))
	}
	return "", ""
}

// markdown renders descriptions as GitHub-flavoured Markdown, the flavour
// EasyYapi renders them with. HTML written in a doc comment, such as a
// Javadoc <p>, is passed through as EasyYapi passes it.
var markdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(goldmarkhtml.WithUnsafe()),
)

// blockBreak matches the line break the renderer writes after a block tag.
// EasyYapi's renderer writes none, so blocks follow each other directly;
// breaks inside a paragraph or a code block are kept.
var blockBreak = regexp.MustCompile(`(</(?:p|h[1-6]|ul|ol|li|blockquote|pre|table|thead|tbody|tr|th|td)>|<(?:ul|ol|blockquote|table|thead|tbody|tr)>|<hr\s*/?>)\n`)

// renderHTML renders a description as YApi's HTML description, the way
// EasyYapi does: Markdown rendered to HTML inside a <body> element. The
// Markdown source travels in the markdown field.
func renderHTML(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	var b bytes.Buffer
	if err := markdown.Convert([]byte(text), &b); err != nil {
		return "<body><p>" + html.EscapeString(text) + "</p></body>"
	}
	return "<body>" + blockBreak.ReplaceAllString(b.String(), "$1") + "</body>"
}

var (
	pathIllegal        = regexp.MustCompile(`[^a-zA-Z0-9\-/_:.{}?=!]`)
	pathSlashRedundant = regexp.MustCompile(`//+`)
)

// normalizePath makes a path YApi accepts: a leading slash, and every
// character YApi rejects replaced by a slash.
func normalizePath(path string) string {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = pathIllegal.ReplaceAllString(path, "/")
	return pathSlashRedundant.ReplaceAllString(path, "/")
}

func flag(required bool) string {
	if required {
		return "1"
	}
	return "0"
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

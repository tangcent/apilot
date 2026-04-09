// Package postman implements the Formatter interface producing Postman Collection v2.1 JSON.
package postman

import (
	"encoding/json"
	"strings"

	"github.com/tangcent/apilot/api-collector/collector"
	"github.com/tangcent/apilot/api-formater/formater"
	"github.com/tangcent/apilot/api-formater-postman/model"
)

const postmanSchema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

// PostmanFormatter formats endpoints as a Postman Collection v2.1 JSON document.
type PostmanFormatter struct{}

// New returns a new PostmanFormatter.
func New() formater.Formatter { return &PostmanFormatter{} }

func (f *PostmanFormatter) Name() string { return "postman" }

func (f *PostmanFormatter) SupportedFormats() []string { return []string{"postman"} }

// Format converts endpoints into a Postman Collection v2.1 JSON document.
// Endpoints are grouped by their Folder field into Postman folders.
// An empty endpoints slice returns a valid empty collection.
func (f *PostmanFormatter) Format(endpoints []collector.ApiEndpoint, opts formater.FormatOptions) ([]byte, error) {
	name := opts.Config["name"]
	if name == "" {
		name = "APilot Export"
	}

	// Group endpoints by folder
	folderMap := map[string][]collector.ApiEndpoint{}
	var folderOrder []string
	for _, ep := range endpoints {
		folder := ep.Folder
		if folder == "" {
			folder = "Default"
		}
		if _, exists := folderMap[folder]; !exists {
			folderOrder = append(folderOrder, folder)
		}
		folderMap[folder] = append(folderMap[folder], ep)
	}

	var groups []model.ItemGroup
	for _, folderName := range folderOrder {
		var items []model.Item
		for _, ep := range folderMap[folderName] {
			items = append(items, buildItem(ep))
		}
		groups = append(groups, model.ItemGroup{Name: folderName, Item: items})
	}

	col := model.Collection{
		Info: model.Info{Name: name, Schema: postmanSchema},
		Item: groups,
	}
	return json.MarshalIndent(col, "", "  ")
}

func buildItem(ep collector.ApiEndpoint) model.Item {
	method := ep.Method
	if method == "" {
		method = "GET"
	}

	var headers []model.Header
	for _, h := range ep.Headers {
		headers = append(headers, model.Header{Key: h.Name, Value: h.Value})
	}

	var queryParams []model.Query
	for _, p := range ep.Parameters {
		if p.In == "query" {
			queryParams = append(queryParams, model.Query{Key: p.Name, Value: ""})
		}
	}

	url := model.URL{
		Raw:   "http://localhost" + ep.Path,
		Path:  splitPath(ep.Path),
		Query: queryParams,
	}

	var body *model.Body
	if ep.RequestBody != nil {
		body = &model.Body{
			Mode: "raw",
			Raw:  "{}",
			Options: &model.BodyOptions{Raw: model.RawOptions{Language: "json"}},
		}
	}

	return model.Item{
		Name: ep.Name,
		Request: model.Request{
			Method: method,
			Header: headers,
			URL:    url,
			Body:   body,
		},
	}
}

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	return parts
}

// Package yapi implements a Formatter that exports endpoints to YApi.
//
// With a YApi server and a project token, the endpoints are saved to that
// project through YApi's open API and the formatter prints what it created,
// updated or failed to save. Without them it prints the interfaces in the JSON
// shape YApi's "data import" accepts, so the export can be checked or imported
// by hand.
package yapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	formatter "github.com/tangcent/apilot/api-formatter"
	model "github.com/tangcent/apilot/api-model"
)

// Params are the formatter's options, given with --params.
type Params struct {
	// Server is the YApi base URL; the yapi.server setting is the default.
	Server string `json:"server"`
	// Token is the YApi project token; the yapi.token setting is the default.
	Token string `json:"token"`
	// Rules names extra EasyYapi rules files (paths or URLs), read after the
	// .easy.api.config files found in the project.
	Rules []string `json:"rules"`
	// SourceDir is where the project's sources are; the engine fills it in.
	SourceDir string `json:"sourceDir"`
}

// YapiFormatter exports endpoints to YApi.
type YapiFormatter struct {
	client *http.Client
}

// New returns a YApi formatter.
func New() formatter.Formatter {
	return &YapiFormatter{client: &http.Client{Timeout: 30 * time.Second}}
}

func (f *YapiFormatter) Name() string { return "yapi" }

func (f *YapiFormatter) RequiredSettings() []formatter.SettingDef {
	return []formatter.SettingDef{
		{
			Key:         "yapi.server",
			Description: "YApi server URL, e.g. https://yapi.example.com; with yapi.token set, the export is saved to YApi",
		},
		{
			Key:         "yapi.token",
			Description: "YApi project token: the project the endpoints are saved to",
		},
	}
}

func (f *YapiFormatter) Format(endpoints []model.ApiEndpoint, opts formatter.FormatOptions) ([]byte, error) {
	var p Params
	if err := opts.DecodeParams(&p); err != nil {
		return nil, err
	}
	server := strings.TrimRight(firstNonEmpty(p.Server, setting(opts, "yapi.server")), "/")
	token := firstNonEmpty(p.Token, setting(opts, "yapi.token"))

	rules, err := LoadRules(p.SourceDir, p.Rules, HTTPFetcher(f.client))
	if err != nil {
		return nil, err
	}
	builder := &schemaBuilder{rules: rules}
	interfaces := make([]Interface, 0, len(endpoints))
	for _, ep := range endpoints {
		interfaces = append(interfaces, buildInterface(ep, builder))
	}

	if server == "" || token == "" {
		return json.MarshalIndent(importDocument(interfaces), "", "  ")
	}

	result, err := upload(&client{server: server, token: token, http: f.client}, interfaces)
	if err != nil {
		return nil, fmt.Errorf("saving to YApi at %s: %w", server, err)
	}
	result.Rules = rules.Sources
	result.Warnings = rules.Warnings
	if result.Failed > 0 {
		var msg strings.Builder
		fmt.Fprintf(&msg, "%d of %d endpoints were not saved to YApi (created: %d, updated: %d, failed: %d)",
			result.Failed, len(interfaces), result.Created, result.Updated, result.Failed)
		for _, item := range result.Items {
			if item.Status == "failed" {
				fmt.Fprintf(&msg, "\n  failed: %s %s: %s", item.Method, item.Path, item.Reason)
			}
		}
		return nil, errors.New(msg.String())
	}
	return json.MarshalIndent(result, "", "  ")
}

// category is one entry of YApi's import format: a category and its
// interfaces.
type category struct {
	Name string      `json:"name"`
	Desc string      `json:"desc"`
	List []Interface `json:"list"`
}

// importDocument groups the interfaces by category, in first-seen order.
func importDocument(interfaces []Interface) []category {
	out := []category{}
	index := map[string]int{}
	for _, it := range interfaces {
		i, ok := index[it.category]
		if !ok {
			i = len(out)
			index[it.category] = i
			out = append(out, category{Name: it.category, List: []Interface{}})
		}
		out[i].List = append(out[i].List, it)
	}
	return out
}

func setting(opts formatter.FormatOptions, key string) string {
	if opts.Settings == nil {
		return ""
	}
	return opts.Settings.Get(key)
}

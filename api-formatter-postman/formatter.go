// Package postman implements the Formatter interface producing Postman Collection v2.1 JSON.
package postman

import (
	"encoding/json"
	"strings"

	"github.com/tangcent/apilot/api-formatter"
	apimodel "github.com/tangcent/apilot/api-model"
	"github.com/tangcent/apilot/api-formatter-postman/model"
)

const postmanSchema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

// Params holds postman-specific formatting options.
type Params struct {
	// CollectionName is the name of the exported Postman collection.
	// Defaults to "APilot Export".
	CollectionName string `json:"collectionName"`

	// BaseURL is prepended to each endpoint path. Defaults to "http://localhost".
	BaseURL string `json:"baseURL"`
}

// PostmanFormatter formats endpoints as a Postman Collection v2.1 JSON document.
type PostmanFormatter struct{}

// New returns a new PostmanFormatter.
func New() formatter.Formatter { return &PostmanFormatter{} }

func (f *PostmanFormatter) Name() string { return "postman" }

// Format converts endpoints into a Postman Collection v2.1 JSON document.
// Endpoints are grouped by their Folder field into Postman folders.
// An empty endpoints slice returns a valid empty collection.
func (f *PostmanFormatter) Format(endpoints []apimodel.ApiEndpoint, opts formatter.FormatOptions) ([]byte, error) {
	var p Params
	if err := opts.DecodeParams(&p); err != nil {
		return nil, err
	}
	if p.CollectionName == "" {
		p.CollectionName = "APilot Export"
	}
	if p.BaseURL == "" {
		p.BaseURL = "http://localhost"
	}

	folderMap := map[string][]apimodel.ApiEndpoint{}
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
			items = append(items, buildItem(ep, p))
		}
		groups = append(groups, model.ItemGroup{Name: folderName, Item: items})
	}

	col := model.Collection{
		Info: model.Info{Name: p.CollectionName, Schema: postmanSchema},
		Item: groups,
	}
	return json.MarshalIndent(col, "", "  ")
}

func buildItem(ep apimodel.ApiEndpoint, p Params) model.Item {
	method := ep.Method
	if method == "" {
		method = "GET"
	}

	var headers []model.Header
	for _, h := range ep.Headers {
		headers = append(headers, model.Header{Key: h.Name, Value: h.Value})
	}

	var queryParams []model.Query
	for _, param := range ep.Parameters {
		if param.In == "query" {
			queryParams = append(queryParams, model.Query{Key: param.Name, Value: ""})
		}
	}

	url := model.URL{
		Raw:   p.BaseURL + ep.Path,
		Path:  splitPath(ep.Path),
		Query: queryParams,
	}

	var body *model.Body
	if ep.RequestBody != nil {
		body = &model.Body{
			Mode:    "raw",
			Raw:     "{}",
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
		Response: buildResponses(ep),
	}
}

func buildResponses(ep apimodel.ApiEndpoint) []model.Response {
	if ep.Response == nil || ep.Response.Example == nil {
		return nil
	}
	bodyBytes, err := json.Marshal(ep.Response.Example)
	if err != nil {
		return nil
	}
	return []model.Response{
		{
			Name:   "Example response",
			Status: "OK",
			Code:   200,
			Body:   string(bodyBytes),
		},
	}
}

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	for i, p := range parts {
		parts[i] = convertPathParam(p)
	}
	return parts
}

func convertPathParam(segment string) string {
	if len(segment) >= 2 && segment[0] == '{' && segment[len(segment)-1] == '}' {
		return ":" + segment[1:len(segment)-1]
	}
	return segment
}

// Package postman implements the Formatter interface producing Postman Collection v2.1 JSON.
package postman

import (
	"encoding/json"
	"fmt"
	"strings"

	formatter "github.com/tangcent/apilot/api-formatter"
	"github.com/tangcent/apilot/api-formatter-postman/model"
	apimodel "github.com/tangcent/apilot/api-model"
)

const postmanSchema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

const (
	ExportModeCreateNew      = "CREATE_NEW"
	ExportModeUpdateExisting = "UPDATE_EXISTING"
)

type Params struct {
	CollectionName string `json:"collectionName"`
	BaseURL        string `json:"baseURL"`
	Mode           string `json:"mode"`
	PostmanAPIKey  string `json:"postmanAPIKey"`
	WorkspaceID    string `json:"workspaceId"`
	CollectionUID  string `json:"collectionUid"`
	ProjectName    string `json:"projectName"`
	ExportMode     string `json:"exportMode"`
	OutputPath     string `json:"outputPath"`
}

type PostmanFormatter struct{}

func New() formatter.Formatter { return &PostmanFormatter{} }

func (f *PostmanFormatter) Name() string { return "postman" }

func (f *PostmanFormatter) RequiredSettings() []formatter.SettingDef {
	return []formatter.SettingDef{
		{
			Key:         "postman.api.key",
			Description: "Postman API key for pushing collections to the Postman API",
			Required:    false,
		},
		{
			Key:         "postman.export.mode",
			Description: "Postman export mode: CREATE_NEW (always create new collection) or UPDATE_EXISTING (update previously exported collection per project)",
			Required:    false,
		},
	}
}

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

	apiKey := resolveAPIKey(p, opts)
	exportMode := resolveExportMode(p, opts)

	col := buildCollection(endpoints, p)

	mode := resolveMode(p, apiKey)

	if mode == "api" {
		return pushToPostmanAPI(apiKey, exportMode, p.ProjectName, p.CollectionUID, p.WorkspaceID, opts.Collections, col)
	}

	return json.MarshalIndent(col, "", "  ")
}

func buildCollection(endpoints []apimodel.ApiEndpoint, p Params) model.Collection {
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

	var items []model.Item
	for _, folderName := range folderOrder {
		var subItems []model.Item
		for _, ep := range folderMap[folderName] {
			subItems = append(subItems, buildItem(ep, p))
		}
		items = append(items, model.Item{
			Name: folderName,
			Item: subItems,
		})
	}

	return model.Collection{
		Info: model.Info{Name: p.CollectionName, Schema: postmanSchema},
		Item: items,
	}
}

func resolveAPIKey(p Params, opts formatter.FormatOptions) string {
	if opts.Settings != nil {
		if key := opts.Settings.Get("postman.api.key"); key != "" {
			return key
		}
	}
	return p.PostmanAPIKey
}

func resolveMode(p Params, apiKey string) string {
	if p.Mode != "" {
		return strings.ToLower(p.Mode)
	}
	if p.OutputPath != "" {
		return "file"
	}
	if apiKey != "" {
		return "api"
	}
	return "file"
}

func resolveExportMode(p Params, opts formatter.FormatOptions) string {
	if p.ExportMode != "" {
		return strings.ToUpper(p.ExportMode)
	}
	if opts.Settings != nil {
		if mode := opts.Settings.Get("postman.export.mode"); mode != "" {
			return strings.ToUpper(mode)
		}
	}
	return ExportModeCreateNew
}

func pushToPostmanAPI(apiKey string, exportMode string, projectName string, paramCollectionUID string, paramWorkspaceID string, store formatter.CollectionStore, col model.Collection) ([]byte, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("postman api key is required for api mode. Set it with: apilot set postman.api.key <value>")
	}
	client := newPostmanClient(apiKey)

	workspaceID := paramWorkspaceID
	collectionUID := paramCollectionUID

	if collectionUID == "" && exportMode == ExportModeUpdateExisting && projectName != "" && store != nil {
		binding, err := store.GetBinding(projectName)
		if err == nil && binding != nil {
			collectionUID = binding.CollectionUID
			if binding.WorkspaceID != "" && workspaceID == "" {
				workspaceID = binding.WorkspaceID
			}
		}
	}

	if collectionUID != "" {
		result, err := client.UpdateCollection(collectionUID, col)
		if err != nil {
			return nil, fmt.Errorf("updating collection in postman api: %w", err)
		}
		if projectName != "" && store != nil {
			_ = store.SetBinding(projectName, formatter.CollectionBinding{
				WorkspaceID:   workspaceID,
				CollectionUID: result.Collection.UID,
			})
		}
		apiResult := APIResult{
			CollectionID:  result.Collection.ID,
			CollectionUID: result.Collection.UID,
			CollectionURL: fmt.Sprintf("https://go.postman.co/collection/%s", result.Collection.UID),
			Action:        "updated",
		}
		return json.MarshalIndent(apiResult, "", "  ")
	}

	result, err := client.CreateCollection(workspaceID, col)
	if err != nil {
		return nil, fmt.Errorf("pushing to postman api: %w", err)
	}

	if projectName != "" && store != nil {
		_ = store.SetBinding(projectName, formatter.CollectionBinding{
			WorkspaceID:   workspaceID,
			CollectionUID: result.Collection.UID,
		})
	}

	apiResult := APIResult{
		CollectionID:  result.Collection.ID,
		CollectionUID: result.Collection.UID,
		CollectionURL: fmt.Sprintf("https://go.postman.co/collection/%s", result.Collection.UID),
		Action:        "created",
	}
	return json.MarshalIndent(apiResult, "", "  ")
}

func buildItem(ep apimodel.ApiEndpoint, p Params) model.Item {
	method := ep.Method
	if method == "" {
		method = "GET"
	}

	var headers []model.Header
	for _, h := range ep.Headers {
		headers = append(headers, model.Header{
			Key:         h.Name,
			Value:       h.Value,
			Type:        "text",
			Description: h.Description,
		})
	}

	queryParams := buildQueryParams(ep.Parameters)
	pathVars := buildPathVars(ep.Parameters)
	pathSegments := splitPath(ep.Path)

	host := []string{p.BaseURL}
	rawURL := p.BaseURL + ep.Path
	if len(queryParams) > 0 {
		var qs []string
		for _, q := range queryParams {
			qs = append(qs, q.Key+"="+q.Value)
		}
		rawURL += "?" + strings.Join(qs, "&")
	}

	url := model.URL{
		Raw:      rawURL,
		Host:     host,
		Path:     pathSegments,
		Query:    queryParams,
		Variable: pathVars,
	}

	body := buildBody(ep)
	req := model.Request{
		Method:      method,
		Header:      headers,
		URL:         url,
		Body:        body,
		Description: ep.Description,
	}

	return model.Item{
		Name:        ep.Name,
		Request:     &req,
		Response:    buildResponses(ep, &req),
		Description: ep.Description,
	}
}

func buildQueryParams(params []apimodel.ApiParameter) []model.Query {
	var queryParams []model.Query
	for _, param := range params {
		if param.In == "query" {
			value := param.Example
			if value == "" {
				value = param.Default
			}
			queryParams = append(queryParams, model.Query{
				Key:         param.Name,
				Value:       value,
				Equals:      true,
				Description: param.Description,
			})
		}
	}
	return queryParams
}

func buildPathVars(params []apimodel.ApiParameter) []model.PathVariable {
	var pathVars []model.PathVariable
	for _, param := range params {
		if param.In == "path" {
			value := param.Example
			if value == "" {
				value = param.Default
			}
			pathVars = append(pathVars, model.PathVariable{
				Key:         param.Name,
				Value:       value,
				Description: param.Description,
			})
		}
	}
	return pathVars
}

func buildBody(ep apimodel.ApiEndpoint) *model.Body {
	if ep.RequestBody == nil {
		return nil
	}

	contentType := strings.ToLower(ep.RequestBody.MediaType)

	if strings.Contains(contentType, "x-www-form-urlencoded") {
		return buildFormBody(ep, "urlencoded")
	}

	if strings.Contains(contentType, "multipart") || strings.Contains(contentType, "form-data") {
		return buildFormBody(ep, "formdata")
	}

	raw := buildRawBody(ep.RequestBody)
	return &model.Body{
		Mode:    "raw",
		Raw:     raw,
		Options: &model.BodyOptions{Raw: model.RawOptions{Language: "json"}},
	}
}

func buildFormBody(ep apimodel.ApiEndpoint, mode string) *model.Body {
	var params []model.FormParam
	for _, param := range ep.Parameters {
		if param.In == "form" {
			value := param.Example
			if value == "" {
				value = param.Default
			}
			paramType := "text"
			if param.Type == "file" {
				paramType = "file"
			}
			params = append(params, model.FormParam{
				Key:         param.Name,
				Value:       value,
				Type:        paramType,
				Description: param.Description,
			})
		}
	}
	body := &model.Body{Mode: mode}
	if mode == "urlencoded" {
		body.Urlencoded = params
	} else {
		body.Formdata = params
	}
	return body
}

func buildRawBody(apiBody *apimodel.ApiBody) string {
	if apiBody.Body != nil {
		return objectModelToJSON(apiBody.Body)
	}
	if apiBody.Example != nil {
		b, err := json.MarshalIndent(apiBody.Example, "", "    ")
		if err != nil {
			return "{}"
		}
		return string(b)
	}
	return "{}"
}

func buildResponses(ep apimodel.ApiEndpoint, req *model.Request) []model.Response {
	if ep.Response == nil {
		return nil
	}

	var bodyStr string
	if ep.Response.Body != nil {
		bodyStr = objectModelToJSON(ep.Response.Body)
	} else if ep.Response.Example != nil {
		b, err := json.MarshalIndent(ep.Response.Example, "", "    ")
		if err != nil {
			bodyStr = ""
		} else {
			bodyStr = string(b)
		}
	}

	var respHeaders []model.Header
	respHeaders = append(respHeaders, model.Header{
		Name:        "content-type",
		Key:         "content-type",
		Value:       "application/json;charset=UTF-8",
		Type:        "text",
		Description: "The mime type of this content",
	})
	respHeaders = append(respHeaders, model.Header{
		Name:        "server",
		Key:         "server",
		Value:       "Apache-Coyote/1.1",
		Type:        "text",
		Description: "A name for the server",
	})

	return []model.Response{
		{
			Name:                   "Example response",
			OriginalRequest:        req,
			Status:                 "OK",
			Code:                   200,
			Header:                 respHeaders,
			Body:                   bodyStr,
			PostmanPreviewLanguage: "json",
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

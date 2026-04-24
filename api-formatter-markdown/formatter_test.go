package markdown

import (
	"encoding/json"
	"strings"
	"testing"

	formatter "github.com/tangcent/apilot/api-formatter"
	model "github.com/tangcent/apilot/api-model"
)

func TestFormat_EmptyInput(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}
	if output == nil {
		t.Fatal("Format() output = nil, want non-nil bytes")
	}
	outputStr := strings.TrimSpace(string(output))
	if outputStr != "" {
		t.Errorf("Format() output = %q, want empty string or whitespace only", outputStr)
	}
}

func TestFormat_SimpleFormat(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:        "Get User",
			Path:        "/users/{id}",
			Method:      "GET",
			Protocol:    "http",
			Description: "Retrieve a user by ID",
			Parameters: []model.ApiParameter{
				{
					Name:        "id",
					In:          "path",
					Type:        "text",
					Required:    true,
					Description: "User ID",
				},
				{
					Name:        "include",
					In:          "query",
					Type:        "text",
					Required:    false,
					Description: "Include related data",
				},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}
	if output == nil {
		t.Fatal("Format() output = nil, want non-nil bytes")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "## GET /users/{id}") {
		t.Error("Output should contain '## GET /users/{id}' headline")
	}
	if !strings.Contains(outputStr, "Retrieve a user by ID") {
		t.Error("Output should contain description")
	}
	if !strings.Contains(outputStr, "| Name | In | Type | Required | Description |") {
		t.Error("Output should contain parameter table header")
	}
	if !strings.Contains(outputStr, "| id | path | text | true | User ID |") {
		t.Error("Output should contain first parameter row")
	}
	if !strings.Contains(outputStr, "| include | query | text | false | Include related data |") {
		t.Error("Output should contain second parameter row")
	}
}

func TestFormat_DetailedFormat(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:        "Create User",
			Path:        "/users",
			Method:      "POST",
			Protocol:    "http",
			Description: "Create a new user",
			Tags:        []string{"users", "admin"},
			Parameters: []model.ApiParameter{
				{
					Name:        "X-Request-ID",
					In:          "header",
					Type:        "text",
					Required:    false,
					Default:     "auto-generated",
					Description: "Request ID for tracing",
				},
			},
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
				Example: map[string]interface{}{
					"name":  "John Doe",
					"email": "john@example.com",
				},
			},
			Response: &model.ApiBody{
				MediaType: "application/json",
				Example: map[string]interface{}{
					"id":     123,
					"name":   "John Doe",
					"email":  "john@example.com",
					"status": "active",
				},
			},
		},
	}
	params := Params{Variant: "detailed"}
	paramsJSON, _ := json.Marshal(params)
	opts := formatter.FormatOptions{Params: paramsJSON}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}
	if output == nil {
		t.Fatal("Format() output = nil, want non-nil bytes")
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "## Create User") {
		t.Error("Output should contain '## Create User' headline")
	}
	if !strings.Contains(outputStr, "**POST** `/users`") {
		t.Error("Output should contain '**POST** `/users`'")
	}
	if !strings.Contains(outputStr, "Create a new user") {
		t.Error("Output should contain description")
	}
	if !strings.Contains(outputStr, "**Tags:** users admin") {
		t.Error("Output should contain tags")
	}
	if !strings.Contains(outputStr, "| Name | In | Type | Required | Default | Description |") {
		t.Error("Output should contain parameter table header with Default column")
	}
	if !strings.Contains(outputStr, "### Request Body") {
		t.Error("Output should contain Request Body section")
	}
	if !strings.Contains(outputStr, "**Media Type:** `application/json`") {
		t.Error("Output should contain media type")
	}
	if !strings.Contains(outputStr, "```json") {
		t.Error("Output should contain JSON code block")
	}
	if !strings.Contains(outputStr, "### Response") {
		t.Error("Output should contain Response section")
	}
}

func TestFormat_SimpleVsDetailed(t *testing.T) {
	f := New()
	endpoint := model.ApiEndpoint{
		Name:        "Update User",
		Path:        "/users/{id}",
		Method:      "PUT",
		Protocol:    "http",
		Description: "Update user information",
		Parameters: []model.ApiParameter{
			{
				Name:        "id",
				In:          "path",
				Type:        "text",
				Required:    true,
				Description: "User ID",
			},
		},
		RequestBody: &model.ApiBody{
			MediaType: "application/json",
			Example:   map[string]interface{}{"name": "Updated Name"},
		},
	}

	simpleOutput, err := f.Format([]model.ApiEndpoint{endpoint}, formatter.FormatOptions{})
	if err != nil {
		t.Fatalf("Simple format error = %v", err)
	}

	params := Params{Variant: "detailed"}
	paramsJSON, _ := json.Marshal(params)
	detailedOutput, err := f.Format([]model.ApiEndpoint{endpoint}, formatter.FormatOptions{Params: paramsJSON})
	if err != nil {
		t.Fatalf("Detailed format error = %v", err)
	}

	simpleStr := string(simpleOutput)
	detailedStr := string(detailedOutput)

	if strings.Contains(simpleStr, "## Update User") {
		t.Error("Simple format should not contain endpoint name as headline")
	}
	if !strings.Contains(simpleStr, "## PUT /users/{id}") {
		t.Error("Simple format should contain '## PUT /users/{id}' headline")
	}
	if strings.Contains(simpleStr, "### Request Body") {
		t.Error("Simple format should not contain Request Body section")
	}

	if !strings.Contains(detailedStr, "## Update User") {
		t.Error("Detailed format should contain endpoint name as headline")
	}
	if !strings.Contains(detailedStr, "**PUT** `/users/{id}`") {
		t.Error("Detailed format should contain '**PUT** `/users/{id}`'")
	}
	if !strings.Contains(detailedStr, "### Request Body") {
		t.Error("Detailed format should contain Request Body section")
	}
}

func TestFormat_WithRequestBody(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:        "Create Order",
			Path:        "/orders",
			Method:      "POST",
			Protocol:    "http",
			Description: "Create a new order",
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
				Example: map[string]interface{}{
					"product_id": 456,
					"quantity":   2,
					"notes":      "Please gift wrap",
				},
			},
		},
	}

	params := Params{Variant: "detailed"}
	paramsJSON, _ := json.Marshal(params)
	opts := formatter.FormatOptions{Params: paramsJSON}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "### Request Body") {
		t.Error("Output should contain Request Body section")
	}
	if !strings.Contains(outputStr, "**Media Type:** `application/json`") {
		t.Error("Output should contain media type")
	}
	if !strings.Contains(outputStr, "```json") {
		t.Error("Output should contain JSON code block")
	}
	if !strings.Contains(outputStr, `"product_id"`) {
		t.Error("Output should contain product_id field in JSON")
	}
	if !strings.Contains(outputStr, `"quantity"`) {
		t.Error("Output should contain quantity field in JSON")
	}
}

func TestFormat_AllFields(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:        "Complete Endpoint",
			Path:        "/api/v1/resource/{id}",
			Method:      "PATCH",
			Protocol:    "http",
			Description: "A complete endpoint with all fields",
			Tags:        []string{"resource", "v1"},
			Parameters: []model.ApiParameter{
				{
					Name:        "id",
					In:          "path",
					Type:        "text",
					Required:    true,
					Default:     "",
					Description: "Resource ID",
					Example:     "123",
					Enum:        []string{"123", "456"},
				},
				{
					Name:        "filter",
					In:          "query",
					Type:        "text",
					Required:    false,
					Default:     "all",
					Description: "Filter type",
				},
			},
			Headers: []model.ApiHeader{
				{
					Name:        "Authorization",
					Value:       "Bearer token",
					Description: "Auth header",
					Required:    true,
				},
			},
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
				Body: model.ObjectModelFrom(map[string]*model.FieldModel{
					"name": {Model: model.SingleModel(model.JsonTypeString)},
				}),
				Example: map[string]interface{}{"name": "Updated Name"},
			},
			Response: &model.ApiBody{
				MediaType: "application/json",
				Example: map[string]interface{}{
					"id":      123,
					"name":    "Updated Name",
					"status":  "success",
					"message": "Resource updated",
				},
			},
		},
	}

	params := Params{Variant: "detailed"}
	paramsJSON, _ := json.Marshal(params)
	opts := formatter.FormatOptions{Params: paramsJSON}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	outputStr := string(output)
	requiredElements := []string{
		"## Complete Endpoint",
		"**PATCH** `/api/v1/resource/{id}`",
		"A complete endpoint with all fields",
		"**Tags:** resource v1",
		"### Parameters",
		"| id | path | text | true |  | Resource ID |",
		"| filter | query | text | false | all | Filter type |",
		"### Request Body",
		"**Media Type:** `application/json`",
		"### Response",
		"```json",
		`"name"`,
		`"status"`,
	}

	for _, elem := range requiredElements {
		if !strings.Contains(outputStr, elem) {
			t.Errorf("Output should contain %q", elem)
		}
	}
}

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
	expected := "# API"
	if outputStr != expected {
		t.Errorf("Format() output = %q, want %q", outputStr, expected)
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
	if !strings.Contains(outputStr, "**Path Params:**") {
		t.Error("Output should contain Path Params section")
	}
	if !strings.Contains(outputStr, "| id |  | YES | User ID |") {
		t.Error("Output should contain path parameter row")
	}
	if !strings.Contains(outputStr, "**Query:**") {
		t.Error("Output should contain Query section")
	}
	if !strings.Contains(outputStr, "| include |  | NO | Include related data |") {
		t.Error("Output should contain query parameter row")
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
	if !strings.Contains(outputStr, "### Create User") {
		t.Error("Output should contain '### Create User' headline")
	}
	if !strings.Contains(outputStr, "**Path:** /users") {
		t.Error("Output should contain '**Path:** /users'")
	}
	if !strings.Contains(outputStr, "**Method:** POST") {
		t.Error("Output should contain '**Method:** POST'")
	}
	if !strings.Contains(outputStr, "Create a new user") {
		t.Error("Output should contain description")
	}
	if !strings.Contains(outputStr, "> BASIC") {
		t.Error("Output should contain '> BASIC' section")
	}
	if !strings.Contains(outputStr, "> REQUEST") {
		t.Error("Output should contain '> REQUEST' section")
	}
	if !strings.Contains(outputStr, "**Headers:**") {
		t.Error("Output should contain Headers section")
	}
	if !strings.Contains(outputStr, "| X-Request-ID | auto-generated | NO | Request ID for tracing |") {
		t.Error("Output should contain header row")
	}
	if !strings.Contains(outputStr, "**Request Demo:**") {
		t.Error("Output should contain Request Demo section")
	}
	if !strings.Contains(outputStr, "```json") {
		t.Error("Output should contain JSON code block")
	}
	if !strings.Contains(outputStr, "> RESPONSE") {
		t.Error("Output should contain '> RESPONSE' section")
	}
	if !strings.Contains(outputStr, "**Response Demo:**") {
		t.Error("Output should contain Response Demo section")
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

	if !strings.Contains(simpleStr, "## PUT /users/{id}") {
		t.Error("Simple format should contain '## PUT /users/{id}' headline")
	}
	if strings.Contains(simpleStr, "### Update User") {
		t.Error("Simple format should not contain endpoint name as headline")
	}
	if strings.Contains(simpleStr, "> REQUEST") {
		t.Error("Simple format should not contain '> REQUEST' section")
	}

	if !strings.Contains(detailedStr, "### Update User") {
		t.Error("Detailed format should contain '### Update User' headline")
	}
	if !strings.Contains(detailedStr, "**Path:** /users/{id}") {
		t.Error("Detailed format should contain '**Path:** /users/{id}'")
	}
	if !strings.Contains(detailedStr, "**Method:** PUT") {
		t.Error("Detailed format should contain '**Method:** PUT'")
	}
	if !strings.Contains(detailedStr, "> REQUEST") {
		t.Error("Detailed format should contain '> REQUEST' section")
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
	if !strings.Contains(outputStr, "> REQUEST") {
		t.Error("Output should contain '> REQUEST' section")
	}
	if !strings.Contains(outputStr, "**Request Demo:**") {
		t.Error("Output should contain Request Demo section")
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
			Folder:      "Resource API",
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
		"## Resource API",
		"### Complete Endpoint",
		"**Path:** /api/v1/resource/{id}",
		"**Method:** PATCH",
		"A complete endpoint with all fields",
		"> BASIC",
		"> REQUEST",
		"**Path Params:**",
		"| id | 123 | YES | Resource ID |",
		"**Query:**",
		"| filter | all | NO | Filter type |",
		"**Headers:**",
		"| Authorization | Bearer token | YES | Auth header |",
		"**Request Body:**",
		"| name | string |  |",
		"**Request Demo:**",
		"> RESPONSE",
		"**Response Demo:**",
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

func TestFormat_GroupByFolder(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Get User",
			Folder:   "User API",
			Path:     "/users/{id}",
			Method:   "GET",
			Protocol: "http",
		},
		{
			Name:     "Create User",
			Folder:   "User API",
			Path:     "/users",
			Method:   "POST",
			Protocol: "http",
		},
		{
			Name:     "Get Order",
			Folder:   "Order API",
			Path:     "/orders/{id}",
			Method:   "GET",
			Protocol: "http",
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
	if !strings.Contains(outputStr, "## Order API") {
		t.Error("Output should contain '## Order API' folder heading")
	}
	if !strings.Contains(outputStr, "## User API") {
		t.Error("Output should contain '## User API' folder heading")
	}
	if !strings.Contains(outputStr, "### Get User") {
		t.Error("Output should contain '### Get User' endpoint")
	}
	if !strings.Contains(outputStr, "### Create User") {
		t.Error("Output should contain '### Create User' endpoint")
	}
	if !strings.Contains(outputStr, "### Get Order") {
		t.Error("Output should contain '### Get Order' endpoint")
	}
}

func TestFormat_BodyTableWithNestedFields(t *testing.T) {
	f := New()

	addressObj := model.NewObjectModelBuilder().
		StringField("street").
		StringField("city").
		Build()

	endpoints := []model.ApiEndpoint{
		{
			Name:     "Create User",
			Path:     "/users",
			Method:   "POST",
			Protocol: "http",
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
				Body: model.NewObjectModelBuilder().
					StringField("name", model.WithComment("user name")).
					ObjectField("address", addressObj, model.WithComment("user address")).
					ArrayField("tags", model.SingleModel(model.JsonTypeString), model.WithComment("user tags")).
					Build(),
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
	if !strings.Contains(outputStr, "| name | string | user name |") {
		t.Error("Output should contain name field row")
	}
	if !strings.Contains(outputStr, "| address | object | user address |") {
		t.Error("Output should contain address field row")
	}
	if !strings.Contains(outputStr, "&ensp;&ensp;&#124;─city") {
		t.Error("Output should contain nested city field with indentation")
	}
	if !strings.Contains(outputStr, "&ensp;&ensp;&#124;─street") {
		t.Error("Output should contain nested street field with indentation")
	}
	if !strings.Contains(outputStr, "| tags | string[] | user tags |") {
		t.Error("Output should contain tags field row")
	}
}

func TestFormat_OutputDemoFalse(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Create User",
			Path:     "/users",
			Method:   "POST",
			Protocol: "http",
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
				Example:   map[string]interface{}{"name": "John"},
			},
			Response: &model.ApiBody{
				MediaType: "application/json",
				Example:   map[string]interface{}{"id": 1},
			},
		},
	}

	outputDemo := false
	params := Params{Variant: "detailed", OutputDemo: &outputDemo}
	paramsJSON, _ := json.Marshal(params)
	opts := formatter.FormatOptions{Params: paramsJSON}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "**Request Demo:**") {
		t.Error("Output should not contain Request Demo when outputDemo is false")
	}
	if strings.Contains(outputStr, "**Response Demo:**") {
		t.Error("Output should not contain Response Demo when outputDemo is false")
	}
	if !strings.Contains(outputStr, "> REQUEST") {
		t.Error("Output should still contain '> REQUEST' section")
	}
	if !strings.Contains(outputStr, "> RESPONSE") {
		t.Error("Output should still contain '> RESPONSE' section")
	}
}

func TestFormat_ModuleName(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Get User",
			Path:     "/users/{id}",
			Method:   "GET",
			Protocol: "http",
		},
	}

	params := Params{Variant: "detailed", ModuleName: "My Service API"}
	paramsJSON, _ := json.Marshal(params)
	opts := formatter.FormatOptions{Params: paramsJSON}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "# My Service API") {
		t.Error("Output should contain custom module name as H1 heading")
	}
}

func TestFormat_GrpcEndpoint(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "GetUser",
			Path:     "/users.UserService/GetUser",
			Method:   "",
			Protocol: "grpc",
			Metadata: map[string]any{
				"serviceName":   "users.UserService",
				"streamingType": "UNARY",
			},
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
				Body: model.NewObjectModelBuilder().
					LongField("id", model.WithComment("user id")).
					Build(),
			},
			Response: &model.ApiBody{
				MediaType: "application/json",
				Body: model.NewObjectModelBuilder().
					LongField("id", model.WithComment("user id")).
					StringField("name", model.WithComment("user name")).
					Build(),
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
	if !strings.Contains(outputStr, "### GetUser") {
		t.Error("Output should contain '### GetUser' headline")
	}
	if !strings.Contains(outputStr, "**Protocol:** gRPC") {
		t.Error("Output should contain gRPC protocol")
	}
	if !strings.Contains(outputStr, "**Service:** users.UserService") {
		t.Error("Output should contain service name")
	}
	if !strings.Contains(outputStr, "**Streaming:** UNARY") {
		t.Error("Output should contain streaming type")
	}
	if !strings.Contains(outputStr, "**Full Path:** /users.UserService/GetUser") {
		t.Error("Output should contain full path")
	}
}

func TestFormat_FormParams(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Upload File",
			Path:     "/upload",
			Method:   "POST",
			Protocol: "http",
			Parameters: []model.ApiParameter{
				{
					Name:        "file",
					In:          "form",
					Type:        "file",
					Required:    true,
					Description: "File to upload",
				},
				{
					Name:        "description",
					In:          "form",
					Type:        "text",
					Required:    false,
					Default:     "",
					Description: "File description",
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
	if !strings.Contains(outputStr, "**Form:**") {
		t.Error("Output should contain Form section")
	}
	if !strings.Contains(outputStr, "| name | value | required | type | desc |") {
		t.Error("Output should contain form table header with type column")
	}
	if !strings.Contains(outputStr, "| file |  | YES | file | File to upload |") {
		t.Error("Output should contain file form parameter row")
	}
	if !strings.Contains(outputStr, "| description |  | NO | text | File description |") {
		t.Error("Output should contain description form parameter row")
	}
}

func TestFormat_FieldWithOptions(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Update Status",
			Path:     "/status",
			Method:   "PUT",
			Protocol: "http",
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
				Body: model.NewObjectModelBuilder().
					StringField("status", model.WithComment("current status"), model.WithRequired(true)).
					Build(),
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
	if !strings.Contains(outputStr, "| status | string | current status |") {
		t.Error("Output should contain status field with comment in desc column")
	}
}

func TestFormat_NoFolderEndpoints(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Health Check",
			Path:     "/health",
			Method:   "GET",
			Protocol: "http",
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
	if !strings.Contains(outputStr, "### Health Check") {
		t.Error("Output should contain endpoint name")
	}
	if !strings.Contains(outputStr, "**Path:** /health") {
		t.Error("Output should contain path")
	}
	if !strings.Contains(outputStr, "**Method:** GET") {
		t.Error("Output should contain method")
	}
	if strings.Contains(outputStr, "## \n") {
		t.Error("Output should not have empty folder heading")
	}
}

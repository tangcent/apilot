package curl

import (
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
	outputStr := strings.TrimSpace(string(output))
	if outputStr != "" {
		t.Errorf("Format() output = %q, want empty string or whitespace only", outputStr)
	}
}

func TestFormat_QueryParams(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Search Users",
			Path:     "/users/search",
			Method:   "GET",
			Protocol: "http",
			Parameters: []model.ApiParameter{
				{
					Name: "q",
					In:   "query",
					Type: "text",
				},
				{
					Name: "limit",
					In:   "query",
					Type: "text",
				},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "curl -X GET") {
		t.Error("Output should contain 'curl -X GET'")
	}
	if !strings.Contains(outputStr, "/users/search?q=&limit=") {
		t.Error("Output should contain query params in URL")
	}
}

func TestFormat_Headers(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Get Profile",
			Path:     "/profile",
			Method:   "GET",
			Protocol: "http",
			Headers: []model.ApiHeader{
				{
					Name:  "Authorization",
					Value: "Bearer token123",
				},
				{
					Name:  "X-API-Key",
					Value: "abc123",
				},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "-H 'Authorization: Bearer token123'") {
		t.Error("Output should contain Authorization header")
	}
	if !strings.Contains(outputStr, "-H 'X-API-Key: abc123'") {
		t.Error("Output should contain X-API-Key header")
	}
}

func TestFormat_RequestBody(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Create User",
			Path:     "/users",
			Method:   "POST",
			Protocol: "http",
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "-H 'Content-Type: application/json'") {
		t.Error("Output should contain Content-Type header")
	}
	if !strings.Contains(outputStr, "-d '{}'") {
		t.Error("Output should contain -d '{}' flag")
	}
}

func TestFormat_FormParams(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Submit Form",
			Path:     "/submit",
			Method:   "POST",
			Protocol: "http",
			Parameters: []model.ApiParameter{
				{
					Name: "username",
					In:   "form",
					Type: "text",
				},
				{
					Name: "password",
					In:   "form",
					Type: "text",
				},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "--data-urlencode 'username='") {
		t.Error("Output should contain --data-urlencode for username")
	}
	if !strings.Contains(outputStr, "--data-urlencode 'password='") {
		t.Error("Output should contain --data-urlencode for password")
	}
}

func TestFormat_PathParams(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Get User",
			Path:     "/users/{id}/posts/{postId}",
			Method:   "GET",
			Protocol: "http",
			Parameters: []model.ApiParameter{
				{
					Name: "id",
					In:   "path",
					Type: "text",
				},
				{
					Name: "postId",
					In:   "path",
					Type: "text",
				},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "/users/<id>/posts/<postId>") {
		t.Error("Output should contain substituted path params")
	}
	if strings.Contains(outputStr, "{id}") {
		t.Error("Output should not contain original path param placeholder {id}")
	}
	if strings.Contains(outputStr, "{postId}") {
		t.Error("Output should not contain original path param placeholder {postId}")
	}
}

func TestFormat_AllFeatures(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Update Post",
			Path:     "/posts/{id}",
			Method:   "PUT",
			Protocol: "http",
			Parameters: []model.ApiParameter{
				{
					Name: "id",
					In:   "path",
					Type: "text",
				},
				{
					Name: "version",
					In:   "query",
					Type: "text",
				},
				{
					Name: "title",
					In:   "form",
					Type: "text",
				},
			},
			Headers: []model.ApiHeader{
				{
					Name:  "Authorization",
					Value: "Bearer token",
				},
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "curl -X PUT") {
		t.Error("Output should contain 'curl -X PUT'")
	}
	if !strings.Contains(outputStr, "/posts/<id>?version=") {
		t.Error("Output should contain substituted path param and query param")
	}
	if !strings.Contains(outputStr, "-H 'Authorization: Bearer token'") {
		t.Error("Output should contain Authorization header")
	}
	if !strings.Contains(outputStr, "--data-urlencode 'title='") {
		t.Error("Output should contain --data-urlencode for form param")
	}
}

func TestFormat_MultipleEndpoints(t *testing.T) {
	f := New()
	endpoints := []model.ApiEndpoint{
		{
			Name:     "Get User",
			Path:     "/users/{id}",
			Method:   "GET",
			Protocol: "http",
			Parameters: []model.ApiParameter{
				{
					Name: "id",
					In:   "path",
					Type: "text",
				},
			},
		},
		{
			Name:     "Create User",
			Path:     "/users",
			Method:   "POST",
			Protocol: "http",
			RequestBody: &model.ApiBody{
				MediaType: "application/json",
			},
		},
	}
	opts := formatter.FormatOptions{}

	output, err := f.Format(endpoints, opts)
	if err != nil {
		t.Fatalf("Format() error = %v, want nil", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "/users/<id>") {
		t.Error("Output should contain substituted path param in first endpoint")
	}
	if !strings.Contains(outputStr, "-d '{}'") {
		t.Error("Output should contain -d '{}' flag in second endpoint")
	}
	parts := strings.Split(outputStr, "\n\n")
	if len(parts) != 2 {
		t.Errorf("Output should have 2 endpoints separated by blank lines, got %d parts", len(parts))
	}
}

package django

import (
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func TestParseBasic(t *testing.T) {
	endpoints, err := Parse("testdata/basic")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints to be extracted, got none")
	}

	foundEndpoints := make(map[string]bool)
	for _, ep := range endpoints {
		key := ep.Name + "_" + ep.Method
		foundEndpoints[key] = true

		if ep.Name == "get_user" {
			if ep.Method != "GET" {
				t.Errorf("get_user: expected method GET, got %s", ep.Method)
			}
			if ep.Path != "/get_user" {
				t.Errorf("get_user: expected path /get_user, got %s", ep.Path)
			}
		}
	}

	expectedEndpoints := []struct {
		name   string
		method string
	}{
		{"get_user", "GET"},
		{"user_list", "GET"},
		{"user_list", "POST"},
		{"user_detail", "PUT"},
		{"user_detail", "DELETE"},
	}

	for _, expected := range expectedEndpoints {
		key := expected.name + "_" + expected.method
		if !foundEndpoints[key] {
			t.Errorf("Expected endpoint %s with method %s not found", expected.name, expected.method)
		}
	}
}

func TestParseAPIView(t *testing.T) {
	endpoints, err := Parse("testdata/apiview")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints to be extracted, got none")
	}

	expectedMethods := map[string][]string{
		"UserList":   {"GET", "POST"},
		"UserDetail": {"GET", "PUT", "DELETE"},
	}

	for className, methods := range expectedMethods {
		for _, method := range methods {
			found := false
			for _, ep := range endpoints {
				if ep.Name == className+"."+strings.ToLower(method) && ep.Method == method {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected endpoint %s.%s not found", className, method)
			}
		}
	}
}

func TestParseViewSet(t *testing.T) {
	endpoints, err := Parse("testdata/viewset")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints to be extracted, got none")
	}

	expectedClasses := []string{"UserViewSet", "PostViewSet"}
	for _, className := range expectedClasses {
		found := false
		for _, ep := range endpoints {
			if strings.Contains(ep.Name, className) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected endpoints for class %s not found", className)
		}
	}
}

func TestParseUrlPatterns(t *testing.T) {
	endpoints, err := Parse("testdata/urlpatterns")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints to be extracted, got none")
	}

	expectedPaths := []string{
		"/users/",
		"/users/{pk}/",
		"/posts/",
	}

	for _, expectedPath := range expectedPaths {
		found := false
		for _, ep := range endpoints {
			if ep.Path == expectedPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected path %s not found", expectedPath)
		}
	}
}

func TestExtractPathParams(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"/users/{id}/", []string{"id"}},
		{"/users/{id}/", []string{"id"}},
		{"/users/{id}/", []string{"id"}},
		{"/users/{user_id}/posts/{post_id}/", []string{"user_id", "post_id"}},
		{"/users/", []string{}},
	}

	for _, test := range tests {
		params := extractPathParams(test.path)
		if len(params) != len(test.expected) {
			t.Errorf("Path %s: expected %d params, got %d", test.path, len(test.expected), len(params))
			continue
		}
		for i, param := range params {
			if param.name != test.expected[i] {
				t.Errorf("Path %s: expected param %s, got %s", test.path, test.expected[i], param.name)
			}
		}
	}
}

func TestConvertRegexToPath(t *testing.T) {
	tests := []struct {
		regex    string
		expected string
	}{
		{"^articles/(?P<year>[0-9]{4})/$", "articles/<year>/"},
		{"^articles/(?P<year>[0-9]{4})/(?P<month>[0-9]{2})/$", "articles/<year>/<month>/"},
		{"^users/\\d+/$", "users/<id>/"},
		{"^posts/\\w+/$", "posts/<name>/"},
	}

	for _, test := range tests {
		result := convertRegexToPath(test.regex)
		if result != test.expected {
			t.Errorf("Regex %s: expected %s, got %s", test.regex, test.expected, result)
		}
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"users/", "/users/"},
		{"users/<int:pk>/", "/users/{pk}/"},
		{"users/<pk>/", "/users/{pk}/"},
		{"posts/<int:post_id>/comments/<int:comment_id>/", "/posts/{post_id}/comments/{comment_id}/"},
		{"/users/", "/users/"},
	}

	for _, test := range tests {
		result := normalizePath(test.input)
		if result != test.expected {
			t.Errorf("Input %s: expected %s, got %s", test.input, test.expected, result)
		}
	}
}

func TestParse_WithSerializers(t *testing.T) {
	endpoints, err := Parse("testdata/serializers")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints to be extracted, got none")
	}

	epMap := make(map[string]collector.ApiEndpoint)
	for _, ep := range endpoints {
		key := ep.Method + " " + ep.Path
		epMap[key] = ep
	}

	t.Run("ViewSet list has response body with UserSerializer schema", func(t *testing.T) {
		ep, ok := epMap["GET /UserViewSet"]
		if !ok {
			t.Fatal("missing GET /UserViewSet endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body for GET /UserViewSet")
		}
		if ep.Response.Body == nil {
			t.Fatal("expected Body for response")
		}
		if ep.Response.Body.Kind != "object" {
			t.Errorf("response body Kind = %q, want %q", ep.Response.Body.Kind, "object")
		}
		if ep.Response.Body.TypeName != "UserSerializer" {
			t.Errorf("response body TypeName = %q, want %q", ep.Response.Body.TypeName, "UserSerializer")
		}
		nameField, ok := ep.Response.Body.Fields["name"]
		if !ok {
			t.Fatal("expected 'name' field in UserSerializer")
		}
		if nameField.Model.TypeName != "string" {
			t.Errorf("UserSerializer.name type = %q, want %q", nameField.Model.TypeName, "string")
		}
	})

	t.Run("ViewSet create has request and response body", func(t *testing.T) {
		ep, ok := epMap["POST /UserViewSet"]
		if !ok {
			t.Fatal("missing POST /UserViewSet endpoint")
		}
		if ep.RequestBody == nil {
			t.Fatal("expected request body for POST /UserViewSet")
		}
		if ep.RequestBody.Body == nil {
			t.Fatal("expected Body for request")
		}
		if ep.RequestBody.Body.TypeName != "UserSerializer" {
			t.Errorf("request body TypeName = %q, want %q", ep.RequestBody.Body.TypeName, "UserSerializer")
		}
		if ep.Response == nil {
			t.Fatal("expected response body for POST /UserViewSet")
		}
	})

	t.Run("APIView GET has response body", func(t *testing.T) {
		ep, ok := epMap["GET /AddressAPIView"]
		if !ok {
			t.Fatal("missing GET /AddressAPIView endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body for GET /AddressAPIView")
		}
		if ep.Response.Body == nil {
			t.Fatal("expected Body for response")
		}
		if ep.Response.Body.TypeName != "AddressSerializer" {
			t.Errorf("response body TypeName = %q, want %q", ep.Response.Body.TypeName, "AddressSerializer")
		}
	})

	t.Run("APIView POST has request and response body", func(t *testing.T) {
		ep, ok := epMap["POST /AddressAPIView"]
		if !ok {
			t.Fatal("missing POST /AddressAPIView endpoint")
		}
		if ep.RequestBody == nil {
			t.Fatal("expected request body for POST /AddressAPIView")
		}
		if ep.Response == nil {
			t.Fatal("expected response body for POST /AddressAPIView")
		}
	})

	t.Run("Nested serializer resolves correctly", func(t *testing.T) {
		ep, ok := epMap["GET /UserWithAddressAPIView"]
		if !ok {
			t.Fatal("missing GET /UserWithAddressAPIView endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body")
		}
		if ep.Response.Body == nil {
			t.Fatal("expected Body for response")
		}
		addrField, ok := ep.Response.Body.Fields["address"]
		if !ok {
			t.Fatal("expected 'address' field in UserWithAddressSerializer")
		}
		if addrField.Model.Kind != "object" {
			t.Errorf("address kind = %q, want %q", addrField.Model.Kind, "object")
		}
		if addrField.Model.TypeName != "AddressSerializer" {
			t.Errorf("address typeName = %q, want %q", addrField.Model.TypeName, "AddressSerializer")
		}
	})

	t.Run("ViewSet destroy has no request/response body", func(t *testing.T) {
		ep, ok := epMap["DELETE /UserViewSet"]
		if !ok {
			t.Fatal("missing DELETE /UserViewSet endpoint")
		}
		if ep.RequestBody != nil {
			t.Error("expected no request body for DELETE")
		}
		if ep.Response != nil {
			t.Error("expected no response body for DELETE")
		}
	})
}

func TestParse_SerializerInheritance(t *testing.T) {
	endpoints, err := Parse("testdata/serializer_inheritance")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints to be extracted, got none")
	}

	epMap := make(map[string]collector.ApiEndpoint)
	for _, ep := range endpoints {
		key := ep.Method + " " + ep.Path
		epMap[key] = ep
	}

	t.Run("ItemSerializer has inherited fields", func(t *testing.T) {
		ep, ok := epMap["GET /ItemViewSet"]
		if !ok {
			t.Fatal("missing GET /ItemViewSet endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body")
		}
		if ep.Response.Body == nil {
			t.Fatal("expected Body for response")
		}
		if ep.Response.Body.TypeName != "ItemSerializer" {
			t.Errorf("response body TypeName = %q, want %q", ep.Response.Body.TypeName, "ItemSerializer")
		}
		if len(ep.Response.Body.Fields) < 4 {
			t.Fatalf("ItemSerializer should have at least 4 fields (2 own + 2 inherited), got %d", len(ep.Response.Body.Fields))
		}
		if _, ok := ep.Response.Body.Fields["id"]; !ok {
			t.Error("expected inherited 'id' field")
		}
		if _, ok := ep.Response.Body.Fields["created_at"]; !ok {
			t.Error("expected inherited 'created_at' field")
		}
		if _, ok := ep.Response.Body.Fields["name"]; !ok {
			t.Error("expected own 'name' field")
		}
		if _, ok := ep.Response.Body.Fields["price"]; !ok {
			t.Error("expected own 'price' field")
		}
	})
}

func TestParse_ModelSerializer(t *testing.T) {
	endpoints, err := Parse("testdata/model_serializer")
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints to be extracted, got none")
	}

	epMap := make(map[string]collector.ApiEndpoint)
	for _, ep := range endpoints {
		key := ep.Method + " " + ep.Path
		epMap[key] = ep
	}

	t.Run("ModelSerializer resolves fields", func(t *testing.T) {
		ep, ok := epMap["GET /ProductViewSet"]
		if !ok {
			t.Fatal("missing GET /ProductViewSet endpoint")
		}
		if ep.Response == nil {
			t.Fatal("expected response body")
		}
		if ep.Response.Body == nil {
			t.Fatal("expected Body for response")
		}
		if ep.Response.Body.TypeName != "ProductSerializer" {
			t.Errorf("response body TypeName = %q, want %q", ep.Response.Body.TypeName, "ProductSerializer")
		}
		nameField, ok := ep.Response.Body.Fields["name"]
		if !ok {
			t.Fatal("expected 'name' field in ProductSerializer")
		}
		if nameField.Model.TypeName != "string" {
			t.Errorf("ProductSerializer.name type = %q, want %q", nameField.Model.TypeName, "string")
		}
		priceField, ok := ep.Response.Body.Fields["price"]
		if !ok {
			t.Fatal("expected 'price' field in ProductSerializer")
		}
		if priceField.Model.TypeName != "float" {
			t.Errorf("ProductSerializer.price type = %q, want %q", priceField.Model.TypeName, "float")
		}
	})
}

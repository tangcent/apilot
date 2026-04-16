package django

import (
	"strings"
	"testing"
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
		"users/",
		"users/<int:pk>/",
		"posts/",
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
		{"users/<int:id>/", []string{"id"}},
		{"users/<id>/", []string{"id"}},
		{"users/{id}/", []string{"id"}},
		{"users/<int:user_id>/posts/<int:post_id>/", []string{"user_id", "post_id"}},
		{"users/", []string{}},
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

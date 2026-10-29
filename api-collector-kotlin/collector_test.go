package kotlincollector

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	javacollector "github.com/tangcent/apilot/api-collector-java"
	model "github.com/tangcent/apilot/api-model"
)

func collectDir(t *testing.T, dir string) ([]collector.ApiEndpoint, *KotlinCollector) {
	t.Helper()
	c := New().(*KotlinCollector)
	abs, err := filepath.Abs(dir)
	if err != nil {
		t.Fatal(err)
	}
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect(%s): %v", dir, err)
	}
	return endpoints, c
}

func findEndpoint(t *testing.T, endpoints []collector.ApiEndpoint, name string) *collector.ApiEndpoint {
	t.Helper()
	for i := range endpoints {
		if endpoints[i].Name == name {
			return &endpoints[i]
		}
	}
	t.Fatalf("endpoint %q not found; got %d endpoints", name, len(endpoints))
	return nil
}

func findParameter(t *testing.T, endpoint *collector.ApiEndpoint, name string, in string) *collector.ApiParameter {
	t.Helper()
	for i := range endpoint.Parameters {
		if endpoint.Parameters[i].Name == name && endpoint.Parameters[i].In == in {
			return &endpoint.Parameters[i]
		}
	}
	t.Fatalf("parameter %q in %q not found on %s", name, in, endpoint.Name)
	return nil
}

func objectField(t *testing.T, object *model.ObjectModel, name string) *model.FieldModel {
	t.Helper()
	if object == nil || object.Kind != model.KindObject {
		t.Fatalf("expected object model, got %#v", object)
	}
	field, ok := object.Fields[name]
	if !ok {
		t.Fatalf("field %q not found; fields=%v", name, fieldNames(object))
	}
	return field
}

func fieldNames(object *model.ObjectModel) []string {
	if object == nil {
		return nil
	}
	names := make([]string, 0, len(object.Fields))
	for name := range object.Fields {
		names = append(names, name)
	}
	return names
}

func dataSchema(t *testing.T, endpoint *collector.ApiEndpoint) *model.ObjectModel {
	t.Helper()
	if endpoint.Response == nil || endpoint.Response.Body == nil {
		t.Fatalf("%s has no response body", endpoint.Name)
	}
	data := objectField(t, endpoint.Response.Body, "data")
	return data.Model
}

func TestCollectKotlinProject(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/project")
	if len(endpoints) != 14 {
		t.Fatalf("expected 14 endpoints, got %d: %v", len(endpoints), endpointNames(endpoints))
	}

	counts := map[string]int{}
	for _, ep := range endpoints {
		counts[ep.Folder]++
	}
	if counts["CategoryController"] != 7 || counts["UserController"] != 7 {
		t.Fatalf("unexpected per-controller distribution: %v", counts)
	}

	t.Run("path, query, header and cookie parameters", func(t *testing.T) {
		ep := findEndpoint(t, endpoints, "getProductsByCategory")
		if ep.Path != "/api/categories/{id}/products" || ep.Method != "GET" {
			t.Fatalf("unexpected endpoint: %s %s", ep.Method, ep.Path)
		}
		id := findParameter(t, ep, "id", "path")
		if id.TypeName != "Long" || !id.Required {
			t.Fatalf("path parameter: %#v", id)
		}
		page := findParameter(t, ep, "pageNum", "query")
		if page.TypeName != "Int" || page.Default != "1" || page.Required {
			t.Fatalf("query parameter: %#v", page)
		}

		search := findEndpoint(t, endpoints, "search")
		status := findParameter(t, search, "status", "query")
		if status.Required {
			t.Fatalf("required=false was not honored: %#v", status)
		}
		if status.TypeName != "Status" {
			t.Fatalf("nullable type leaked into typeName: %q", status.TypeName)
		}
		if len(search.Headers) != 1 || search.Headers[0].Name != "X-Tenant" {
			t.Fatalf("header name was not taken from the annotation: %#v", search.Headers)
		}
		findParameter(t, search, "sessionId", "cookie")
	})

	t.Run("request body comes from the parameter type, not its name", func(t *testing.T) {
		ep := findEndpoint(t, endpoints, "create")
		if ep.RequestBody == nil || ep.RequestBody.Body == nil {
			t.Fatal("missing request body schema")
		}
		if ep.RequestBody.Body.TypeName != "CreateCategoryReq" {
			t.Fatalf("request body type = %q", ep.RequestBody.Body.TypeName)
		}
		if _, ok := ep.RequestBody.Body.Fields["req"]; ok {
			t.Fatal("parameter name leaked in as a wrapper object")
		}

		name := objectField(t, ep.RequestBody.Body, "name")
		if name.Model.TypeName != model.JsonTypeString || !name.Required {
			t.Fatalf("name field: %#v", name)
		}
		parent := objectField(t, ep.RequestBody.Body, "parentId")
		if parent.Required {
			t.Fatal("@Nullable must make a field optional")
		}
		if parent.Model.Kind != model.KindSingle || parent.Model.TypeName != model.JsonTypeLong {
			t.Fatalf("parentId model: %#v", parent.Model)
		}
		if parent.DefaultValue != "" {
			t.Fatalf("Kotlin default `= null` leaked into defaultValue: %q", parent.DefaultValue)
		}

		display := objectField(t, ep.RequestBody.Body, "display_name")
		// Nullability alone does not imply optional; this property has no
		// optionality annotation, so it stays required.
		if !display.Required {
			t.Fatalf("display_name should be required: %#v", display)
		}
		if _, ok := ep.RequestBody.Body.Fields["displayName"]; ok {
			t.Fatal("@JsonProperty rename was not applied")
		}
		if _, ok := ep.RequestBody.Body.Fields["secret"]; ok {
			t.Fatal("@JsonIgnore field was exported")
		}
		sortField := objectField(t, ep.RequestBody.Body, "sort")
		if sortField.Comment != "sort order" || sortField.Demo != "1" {
			t.Fatalf("schema documentation missing: %#v", sortField)
		}
		if len(sortField.Options) != 0 {
			t.Fatalf("unexpected options: %#v", sortField.Options)
		}
	})

	t.Run("Result<Void> renders data:null", func(t *testing.T) {
		ep := findEndpoint(t, endpoints, "deleteUser")
		data := dataSchema(t, ep)
		if data.Kind != model.KindSingle || data.TypeName != model.JsonTypeNull {
			t.Fatalf("data model = %#v", data)
		}
	})

	t.Run("Result<PageResult<Any>> nests and Any becomes an object", func(t *testing.T) {
		ep := findEndpoint(t, endpoints, "listOrders")
		data := dataSchema(t, ep)
		if data.Kind != model.KindObject || data.TypeName != "PageResult" {
			t.Fatalf("data model = %#v", data)
		}
		items := objectField(t, data, "items")
		if items.Model.Kind != model.KindArray || items.Model.Items.Kind != model.KindObject {
			t.Fatalf("items model = %#v", items.Model)
		}
	})

	t.Run("recursive and cross-file types resolve", func(t *testing.T) {
		ep := findEndpoint(t, endpoints, "getCategoryTree")
		data := dataSchema(t, ep)
		if data.Kind != model.KindArray || data.Items.TypeName != "CategoryTreeVO" {
			t.Fatalf("tree data model = %#v", data)
		}
		children := objectField(t, data.Items, "children")
		if children.Model.Kind != model.KindArray || children.Model.Items.Kind != model.KindRef {
			t.Fatalf("recursive children model = %#v", children.Model)
		}
		if children.Model.Items.TypeName != "CategoryTreeVO" {
			t.Fatalf("recursive reference name = %q", children.Model.Items.TypeName)
		}

		user := findEndpoint(t, endpoints, "getUser")
		userData := dataSchema(t, user)
		if userData.TypeName != "User" {
			t.Fatalf("cross-file User did not resolve: %#v", userData)
		}
		if _, ok := userData.Fields["createdAt"]; !ok {
			t.Fatalf("inherited fields missing: %v", fieldNames(userData))
		}
	})

	t.Run("enum becomes a string with options", func(t *testing.T) {
		ep := findEndpoint(t, endpoints, "updateSort")
		data := dataSchema(t, ep)
		status := objectField(t, data, "status")
		if status.Model.TypeName != model.JsonTypeString {
			t.Fatalf("enum model = %#v", status.Model)
		}
		if len(status.Options) != 2 || status.Options[0].Value != "ACTIVE" || status.Options[1].Value != "DISABLED" {
			t.Fatalf("enum options = %#v", status.Options)
		}
		if _, ok := c.Unresolved()["Status"]; ok {
			t.Fatal("enum was reported as an unresolved type")
		}
	})

	t.Run("data class does not leak generated members", func(t *testing.T) {
		ep := findEndpoint(t, endpoints, "getUser")
		user := dataSchema(t, ep)
		for _, name := range fieldNames(user) {
			switch name {
			case "component1", "component2", "copy", "toString", "hashCode", "equals":
				t.Fatalf("generated member %q leaked as a field", name)
			}
		}
	})

	t.Run("multipart file parameter", func(t *testing.T) {
		ep := findEndpoint(t, endpoints, "upload")
		file := findParameter(t, ep, "file", "form")
		if file.Type != "file" {
			t.Fatalf("file parameter = %#v", file)
		}
	})
}

func TestUseSiteTargetIsRejected(t *testing.T) {
	c := New().(*KotlinCollector)
	abs, err := filepath.Abs("testdata/invalid-usecase")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err == nil {
		t.Fatal("expected an error for a use-site target")
	}
	message := err.Error()
	if !strings.Contains(message, "InvalidController.kt") {
		t.Fatalf("error does not name the file: %s", message)
	}
	if !strings.Contains(message, "@get:") || !strings.Contains(message, "JsonProperty") {
		t.Fatalf("error does not name the construct: %s", message)
	}
}

func TestZeroEndpointsIsAnEmptySuccess(t *testing.T) {
	endpoints, _ := collectDir(t, "testdata/empty")
	if len(endpoints) != 0 {
		t.Fatalf("expected no endpoints, got %d", len(endpoints))
	}
}

func TestGoldenEndpointsMatchJava(t *testing.T) {
	kotlinEndpoints, _ := collectDir(t, "testdata/golden/kotlin")

	javaDir, err := filepath.Abs("../samples/java-springmvc")
	if err != nil {
		t.Fatal(err)
	}
	javaCollector := javacollector.New()
	javaEndpoints, err := javaCollector.Collect(collector.CollectContext{SourceDir: javaDir, NoDeps: true})
	if err != nil {
		t.Fatalf("java collection failed: %v", err)
	}

	for _, name := range []string{"getUser", "deleteUser", "getProfile", "checkUsername"} {
		kotlinEndpoint := findEndpoint(t, kotlinEndpoints, name)
		javaEndpoint := findEndpoint(t, javaEndpoints, name)
		kotlinJSON, err := json.Marshal(kotlinEndpoint)
		if err != nil {
			t.Fatal(err)
		}
		javaJSON, err := json.Marshal(javaEndpoint)
		if err != nil {
			t.Fatal(err)
		}
		if string(kotlinJSON) != string(javaJSON) {
			t.Fatalf("%s does not match Java\nkotlin: %s\njava:   %s", name, kotlinJSON, javaJSON)
		}
	}
}

func endpointNames(endpoints []collector.ApiEndpoint) []string {
	names := make([]string, 0, len(endpoints))
	for _, ep := range endpoints {
		names = append(names, ep.Name)
	}
	return names
}

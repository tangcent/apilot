package express

import (
	"testing"

	model "github.com/tangcent/apilot/api-model"
)

func TestTSTypeResolver_Primitives(t *testing.T) {
	registry := NewTSTypeRegistry()
	resolver := NewTSTypeResolver(registry)

	tests := []struct {
		input    string
		kind     model.ObjectModelKind
		typeName string
	}{
		{"string", model.KindSingle, model.JsonTypeString},
		{"number", model.KindSingle, model.JsonTypeInt},
		{"boolean", model.KindSingle, model.JsonTypeBoolean},
		{"null", model.KindSingle, model.JsonTypeNull},
		{"void", model.KindSingle, model.JsonTypeNull},
		{"any", model.KindSingle, model.JsonTypeString},
		{"unknown", model.KindSingle, model.JsonTypeString},
	}

	for _, tt := range tests {
		result := resolver.Resolve(tt.input, nil)
		if result.Kind != tt.kind {
			t.Errorf("Resolve(%q).Kind = %q, want %q", tt.input, result.Kind, tt.kind)
		}
		if result.TypeName != tt.typeName {
			t.Errorf("Resolve(%q).TypeName = %q, want %q", tt.input, result.TypeName, tt.typeName)
		}
	}
}

func TestTSTypeResolver_ArrayTypes(t *testing.T) {
	registry := NewTSTypeRegistry()
	resolver := NewTSTypeResolver(registry)

	result := resolver.Resolve("string[]", nil)
	if !result.IsArray() {
		t.Errorf("Resolve('string[]') should be array, got %s", result.Kind)
	}
	if result.Items == nil || result.Items.TypeName != model.JsonTypeString {
		t.Errorf("Resolve('string[]').Items should be string, got %v", result.Items)
	}

	result = resolver.Resolve("number[]", nil)
	if !result.IsArray() {
		t.Errorf("Resolve('number[]') should be array, got %s", result.Kind)
	}
}

func TestTSTypeResolver_GenericArray(t *testing.T) {
	registry := NewTSTypeRegistry()
	resolver := NewTSTypeResolver(registry)

	result := resolver.Resolve("Array<string>", nil)
	if !result.IsArray() {
		t.Errorf("Resolve('Array<string>') should be array, got %s", result.Kind)
	}
	if result.Items == nil || result.Items.TypeName != model.JsonTypeString {
		t.Errorf("Resolve('Array<string>').Items should be string, got %v", result.Items)
	}
}

func TestTSTypeResolver_InterfaceResolution(t *testing.T) {
	registry := NewTSTypeRegistry()
	registry.Interfaces["User"] = &TSInterface{
		Name: "User",
		Fields: []TSField{
			{Name: "id", Type: "number", Required: true},
			{Name: "name", Type: "string", Required: true},
			{Name: "email", Type: "string", Required: true},
			{Name: "age", Type: "number", Required: false},
		},
	}

	resolver := NewTSTypeResolver(registry)
	result := resolver.Resolve("User", nil)

	if !result.IsObject() {
		t.Fatalf("Resolve('User') should be object, got %s", result.Kind)
	}
	if result.TypeName != "User" {
		t.Errorf("Resolve('User').TypeName = %q, want 'User'", result.TypeName)
	}
	if len(result.Fields) != 4 {
		t.Fatalf("Resolve('User') should have 4 fields, got %d", len(result.Fields))
	}

	nameField, ok := result.Fields["name"]
	if !ok {
		t.Fatal("User should have 'name' field")
	}
	if nameField.Model.TypeName != model.JsonTypeString {
		t.Errorf("User.name type = %q, want %q", nameField.Model.TypeName, model.JsonTypeString)
	}
	if !nameField.Required {
		t.Error("User.name should be required")
	}

	ageField, ok := result.Fields["age"]
	if !ok {
		t.Fatal("User should have 'age' field")
	}
	if ageField.Required {
		t.Error("User.age should be optional")
	}
}

func TestTSTypeResolver_TypeAliasResolution(t *testing.T) {
	registry := NewTSTypeRegistry()
	registry.Interfaces["User"] = &TSInterface{
		Name: "User",
		Fields: []TSField{
			{Name: "id", Type: "number", Required: true},
			{Name: "name", Type: "string", Required: true},
		},
	}
	registry.TypeAliases["UserList"] = &TSTypeAlias{
		Name:    "UserList",
		TypeDef: "Array<User>",
	}

	resolver := NewTSTypeResolver(registry)
	result := resolver.Resolve("UserList", nil)

	if !result.IsArray() {
		t.Fatalf("Resolve('UserList') should be array, got %s", result.Kind)
	}
	if result.Items == nil || !result.Items.IsObject() {
		t.Errorf("Resolve('UserList').Items should be object, got %v", result.Items)
	}
}

func TestTSTypeResolver_GenericInterface(t *testing.T) {
	registry := NewTSTypeRegistry()
	registry.Interfaces["PaginatedResponse"] = &TSInterface{
		Name:           "PaginatedResponse",
		TypeParameters: []string{"T"},
		Fields: []TSField{
			{Name: "items", Type: "Array<T>", Required: true},
			{Name: "total", Type: "number", Required: true},
			{Name: "page", Type: "number", Required: true},
		},
	}
	registry.Interfaces["User"] = &TSInterface{
		Name: "User",
		Fields: []TSField{
			{Name: "id", Type: "number", Required: true},
			{Name: "name", Type: "string", Required: true},
		},
	}

	resolver := NewTSTypeResolver(registry)
	result := resolver.Resolve("PaginatedResponse<User>", nil)

	if !result.IsObject() {
		t.Fatalf("Resolve('PaginatedResponse<User>') should be object, got %s", result.Kind)
	}

	itemsField, ok := result.Fields["items"]
	if !ok {
		t.Fatal("PaginatedResponse<User> should have 'items' field")
	}
	if !itemsField.Model.IsArray() {
		t.Errorf("PaginatedResponse<User>.items should be array, got %s", itemsField.Model.Kind)
	}
	if itemsField.Model.Items == nil || !itemsField.Model.Items.IsObject() {
		t.Errorf("PaginatedResponse<User>.items items should be object, got %v", itemsField.Model.Items)
	}
}

func TestTSTypeResolver_UnionType(t *testing.T) {
	registry := NewTSTypeRegistry()
	resolver := NewTSTypeResolver(registry)

	result := resolver.Resolve("string | null", nil)
	if !result.IsSingle() {
		t.Errorf("Resolve('string | null') should be single, got %s", result.Kind)
	}
	if result.TypeName != model.JsonTypeString {
		t.Errorf("Resolve('string | null').TypeName = %q, want %q", result.TypeName, model.JsonTypeString)
	}
}

func TestTSTypeResolver_IntersectionType(t *testing.T) {
	registry := NewTSTypeRegistry()
	registry.Interfaces["Named"] = &TSInterface{
		Name: "Named",
		Fields: []TSField{
			{Name: "name", Type: "string", Required: true},
		},
	}
	registry.Interfaces["Aged"] = &TSInterface{
		Name: "Aged",
		Fields: []TSField{
			{Name: "age", Type: "number", Required: true},
		},
	}

	resolver := NewTSTypeResolver(registry)
	result := resolver.Resolve("Named & Aged", nil)

	if !result.IsObject() {
		t.Fatalf("Resolve('Named & Aged') should be object, got %s", result.Kind)
	}
	if len(result.Fields) != 2 {
		t.Errorf("Resolve('Named & Aged') should have 2 fields, got %d", len(result.Fields))
	}
	if _, ok := result.Fields["name"]; !ok {
		t.Error("Named & Aged should have 'name' field")
	}
	if _, ok := result.Fields["age"]; !ok {
		t.Error("Named & Aged should have 'age' field")
	}
}

func TestTSTypeResolver_RecordType(t *testing.T) {
	registry := NewTSTypeRegistry()
	resolver := NewTSTypeResolver(registry)

	result := resolver.Resolve("Record<string, number>", nil)
	if !result.IsMap() {
		t.Errorf("Resolve('Record<string, number>') should be map, got %s", result.Kind)
	}
}

func TestTSTypeResolver_PromiseType(t *testing.T) {
	registry := NewTSTypeRegistry()
	registry.Interfaces["User"] = &TSInterface{
		Name: "User",
		Fields: []TSField{
			{Name: "id", Type: "number", Required: true},
		},
	}

	resolver := NewTSTypeResolver(registry)
	result := resolver.Resolve("Promise<User>", nil)

	if !result.IsObject() {
		t.Errorf("Resolve('Promise<User>') should unwrap to object, got %s", result.Kind)
	}
}

func TestTSTypeResolver_CircularReference(t *testing.T) {
	registry := NewTSTypeRegistry()
	registry.Interfaces["Node"] = &TSInterface{
		Name: "Node",
		Fields: []TSField{
			{Name: "value", Type: "string", Required: true},
			{Name: "children", Type: "Array<Node>", Required: false},
		},
	}

	resolver := NewTSTypeResolver(registry)
	result := resolver.Resolve("Node", nil)

	if !result.IsObject() {
		t.Fatalf("Resolve('Node') should be object, got %s", result.Kind)
	}

	childrenField, ok := result.Fields["children"]
	if !ok {
		t.Fatal("Node should have 'children' field")
	}
	if !childrenField.Model.IsArray() {
		t.Errorf("Node.children should be array, got %s", childrenField.Model.Kind)
	}
}

func TestTSTypeResolver_UnknownType(t *testing.T) {
	registry := NewTSTypeRegistry()
	resolver := NewTSTypeResolver(registry)

	result := resolver.Resolve("UnknownType", nil)
	if !result.IsSingle() {
		t.Errorf("Resolve('UnknownType') should be single, got %s", result.Kind)
	}
	if result.TypeName != "UnknownType" {
		t.Errorf("Resolve('UnknownType').TypeName = %q, want 'UnknownType'", result.TypeName)
	}
}

func TestTSTypeResolver_EnumResolution(t *testing.T) {
	registry := NewTSTypeRegistry()
	registry.Enums["Status"] = &TSEnum{
		Name: "Status",
		Members: []TSEnumMember{
			{Name: "Active", Value: ""},
			{Name: "Inactive", Value: ""},
		},
	}

	resolver := NewTSTypeResolver(registry)
	result := resolver.Resolve("Status", nil)

	if !result.IsSingle() {
		t.Errorf("Resolve('Status') should be single, got %s", result.Kind)
	}
	if result.TypeName != "Status" {
		t.Errorf("Resolve('Status').TypeName = %q, want 'Status'", result.TypeName)
	}
}

func TestTSTypeResolver_TypeBindings(t *testing.T) {
	registry := NewTSTypeRegistry()
	registry.Interfaces["User"] = &TSInterface{
		Name: "User",
		Fields: []TSField{
			{Name: "id", Type: "number", Required: true},
		},
	}

	resolver := NewTSTypeResolver(registry)
	result := resolver.Resolve("T", map[string]string{"T": "User"})

	if !result.IsObject() {
		t.Errorf("Resolve('T' with binding T->User) should be object, got %s", result.Kind)
	}
	if result.TypeName != "User" {
		t.Errorf("Resolve('T' with binding T->User).TypeName = %q, want 'User'", result.TypeName)
	}
}

func TestParseExpressRequestGenerics(t *testing.T) {
	tests := []struct {
		input           string
		wantReqBody     string
		wantQuery       string
		wantParams      string
	}{
		{
			"Request<{}, {}, CreateUserRequest>",
			"CreateUserRequest", "", "",
		},
		{
			"Request<{ id: string }>",
			"", "", "{ id: string }",
		},
		{
			"Request<{ id: string }, {}, CreateUserRequest>",
			"CreateUserRequest", "", "{ id: string }",
		},
		{
			"Request<{ id: string }, {}, CreateUserRequest, QueryParams>",
			"CreateUserRequest", "QueryParams", "{ id: string }",
		},
		{
			"Request",
			"", "", "",
		},
		{
			"Request<ParamsDictionary, {}, CreateUserBody, ParsedQs>",
			"CreateUserBody", "", "",
		},
	}

	for _, tt := range tests {
		reqBody, query, params := parseExpressRequestGenerics(tt.input)
		if reqBody != tt.wantReqBody {
			t.Errorf("parseExpressRequestGenerics(%q) reqBody = %q, want %q", tt.input, reqBody, tt.wantReqBody)
		}
		if query != tt.wantQuery {
			t.Errorf("parseExpressRequestGenerics(%q) query = %q, want %q", tt.input, query, tt.wantQuery)
		}
		if params != tt.wantParams {
			t.Errorf("parseExpressRequestGenerics(%q) params = %q, want %q", tt.input, params, tt.wantParams)
		}
	}
}

func TestSplitTypeArgs(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"string, number", []string{"string", "number"}},
		{"Array<User>", []string{"Array<User>"}},
		{"string, Array<User>", []string{"string", "Array<User>"}},
		{"string, number, boolean", []string{"string", "number", "boolean"}},
	}

	for _, tt := range tests {
		result := splitTypeArgs(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitTypeArgs(%q) = %v, want %v", tt.input, result, tt.expected)
			continue
		}
		for i, v := range result {
			if v != tt.expected[i] {
				t.Errorf("splitTypeArgs(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
			}
		}
	}
}

func TestParseExpressResponseGenerics(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Response<UserResponse>", "UserResponse"},
		{"Response", ""},
		{"Response<any>", ""},
		{"Response<{}>", ""},
		{"Response<ListUsersResponse>", "ListUsersResponse"},
		{"Response<PaginatedResponse<User>>", "PaginatedResponse<User>"},
	}

	for _, tt := range tests {
		result := parseExpressResponseGenerics(tt.input)
		if result != tt.expected {
			t.Errorf("parseExpressResponseGenerics(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestResolveHandlerTypes(t *testing.T) {
	registry := NewTSTypeRegistry()
	registry.Interfaces["CreateUserRequest"] = &TSInterface{
		Name: "CreateUserRequest",
		Fields: []TSField{
			{Name: "name", Type: "string", Required: true},
			{Name: "email", Type: "string", Required: true},
		},
	}
	registry.Interfaces["UserResponse"] = &TSInterface{
		Name: "UserResponse",
		Fields: []TSField{
			{Name: "id", Type: "number", Required: true},
			{Name: "name", Type: "string", Required: true},
		},
	}

	handlerInfo := &ExpressHandlerInfo{
		ReqBodyType: "CreateUserRequest",
		ResBodyType: "UserResponse",
	}

	reqBody, resBody := ResolveHandlerTypes(handlerInfo, registry)

	if reqBody == nil || !reqBody.IsObject() {
		t.Errorf("reqBody should be object, got %v", reqBody)
	} else {
		if _, ok := reqBody.Fields["name"]; !ok {
			t.Error("reqBody should have 'name' field")
		}
	}

	if resBody == nil || !resBody.IsObject() {
		t.Errorf("resBody should be object, got %v", resBody)
	} else {
		if _, ok := resBody.Fields["id"]; !ok {
			t.Error("resBody should have 'id' field")
		}
	}
}

package django

import (
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	model "github.com/tangcent/apilot/api-model"
)

func TestExtractSerializers_BasicSerializer(t *testing.T) {
	source := []byte(`
from rest_framework import serializers

class UserSerializer(serializers.Serializer):
    name = serializers.CharField(max_length=100)
    email = serializers.EmailField()
    age = serializers.IntegerField(required=False)
`)

	p := createTestParser(t)
	tree := p.Parse(source, nil)
	defer tree.Close()

	rootNode := tree.RootNode()
	serializers := extractSerializers(rootNode, source)

	if len(serializers) != 1 {
		t.Fatalf("expected 1 serializer, got %d", len(serializers))
	}

	userSer, ok := serializers["UserSerializer"]
	if !ok {
		t.Fatal("expected 'UserSerializer' to be found")
	}
	if len(userSer.Fields) != 3 {
		t.Fatalf("UserSerializer fields count = %d, want 3", len(userSer.Fields))
	}

	nameField := findSerializerField(userSer.Fields, "name")
	if nameField == nil {
		t.Fatal("expected 'name' field")
	}
	if nameField.DRFType != "CharField" {
		t.Errorf("name.DRFType = %q, want %q", nameField.DRFType, "CharField")
	}
	if !nameField.Required {
		t.Errorf("name.Required = false, want true")
	}

	ageField := findSerializerField(userSer.Fields, "age")
	if ageField == nil {
		t.Fatal("expected 'age' field")
	}
	if ageField.Required {
		t.Errorf("age.Required = true, want false (required=False)")
	}
}

func TestExtractSerializers_NestedSerializer(t *testing.T) {
	source := []byte(`
from rest_framework import serializers

class AddressSerializer(serializers.Serializer):
    street = serializers.CharField()
    city = serializers.CharField()

class UserWithAddressSerializer(serializers.Serializer):
    name = serializers.CharField()
    address = AddressSerializer()
`)

	p := createTestParser(t)
	tree := p.Parse(source, nil)
	defer tree.Close()

	rootNode := tree.RootNode()
	serializers := extractSerializers(rootNode, source)

	if len(serializers) != 2 {
		t.Fatalf("expected 2 serializers, got %d", len(serializers))
	}

	userSer, ok := serializers["UserWithAddressSerializer"]
	if !ok {
		t.Fatal("expected 'UserWithAddressSerializer'")
	}

	addrField := findSerializerField(userSer.Fields, "address")
	if addrField == nil {
		t.Fatal("expected 'address' field")
	}
	if addrField.DRFType != "AddressSerializer" {
		t.Errorf("address.DRFType = %q, want %q", addrField.DRFType, "AddressSerializer")
	}
}

func TestExtractSerializers_ModelSerializer(t *testing.T) {
	source := []byte(`
from rest_framework import serializers

class ProductSerializer(serializers.ModelSerializer):
    name = serializers.CharField()
    price = serializers.FloatField()

    class Meta:
        model = 'Product'
        fields = ['id', 'name', 'price']
`)

	p := createTestParser(t)
	tree := p.Parse(source, nil)
	defer tree.Close()

	rootNode := tree.RootNode()
	serializers := extractSerializers(rootNode, source)

	if len(serializers) != 1 {
		t.Fatalf("expected 1 serializer, got %d", len(serializers))
	}

	prodSer, ok := serializers["ProductSerializer"]
	if !ok {
		t.Fatal("expected 'ProductSerializer'")
	}
	if !prodSer.IsModelSerializer {
		t.Error("ProductSerializer.IsModelSerializer = false, want true")
	}
	if prodSer.MetaModel != "Product" {
		t.Errorf("ProductSerializer.MetaModel = %q, want %q", prodSer.MetaModel, "Product")
	}
	if len(prodSer.MetaFields) != 3 {
		t.Fatalf("ProductSerializer.MetaFields count = %d, want 3", len(prodSer.MetaFields))
	}
}

func TestExtractSerializers_SerializerInheritance(t *testing.T) {
	source := []byte(`
from rest_framework import serializers

class BaseSerializer(serializers.Serializer):
    id = serializers.IntegerField(read_only=True)
    created_at = serializers.DateTimeField(read_only=True)

class ItemSerializer(BaseSerializer):
    name = serializers.CharField()
    price = serializers.FloatField()
`)

	p := createTestParser(t)
	tree := p.Parse(source, nil)
	defer tree.Close()

	rootNode := tree.RootNode()
	serializers := extractSerializers(rootNode, source)

	if len(serializers) != 2 {
		t.Fatalf("expected 2 serializers, got %d", len(serializers))
	}

	itemSer, ok := serializers["ItemSerializer"]
	if !ok {
		t.Fatal("expected 'ItemSerializer'")
	}
	if len(itemSer.EmbeddedTypes) != 1 || itemSer.EmbeddedTypes[0] != "BaseSerializer" {
		t.Errorf("ItemSerializer.EmbeddedTypes = %v, want [BaseSerializer]", itemSer.EmbeddedTypes)
	}
}

func TestExtractSerializers_ReadOnlyField(t *testing.T) {
	source := []byte(`
from rest_framework import serializers

class ReadOnlySerializer(serializers.Serializer):
    id = serializers.IntegerField(read_only=True)
    name = serializers.CharField()
`)

	p := createTestParser(t)
	tree := p.Parse(source, nil)
	defer tree.Close()

	rootNode := tree.RootNode()
	serializers := extractSerializers(rootNode, source)

	ser, ok := serializers["ReadOnlySerializer"]
	if !ok {
		t.Fatal("expected 'ReadOnlySerializer'")
	}

	idField := findSerializerField(ser.Fields, "id")
	if idField == nil {
		t.Fatal("expected 'id' field")
	}
	if !idField.ReadOnly {
		t.Error("id.ReadOnly = false, want true")
	}
}

func TestExtractSerializers_ManyField(t *testing.T) {
	source := []byte(`
from rest_framework import serializers

class TagSerializer(serializers.Serializer):
    name = serializers.CharField()

class ArticleSerializer(serializers.Serializer):
    title = serializers.CharField()
    tags = TagSerializer(many=True)
`)

	p := createTestParser(t)
	tree := p.Parse(source, nil)
	defer tree.Close()

	rootNode := tree.RootNode()
	serializers := extractSerializers(rootNode, source)

	ser, ok := serializers["ArticleSerializer"]
	if !ok {
		t.Fatal("expected 'ArticleSerializer'")
	}

	tagsField := findSerializerField(ser.Fields, "tags")
	if tagsField == nil {
		t.Fatal("expected 'tags' field")
	}
	if !tagsField.Many {
		t.Error("tags.Many = false, want true")
	}
}

func TestExtractSerializers_DefaultValue(t *testing.T) {
	source := []byte(`
from rest_framework import serializers

class ConfigSerializer(serializers.Serializer):
    name = serializers.CharField(default="unnamed")
    value = serializers.CharField()
`)

	p := createTestParser(t)
	tree := p.Parse(source, nil)
	defer tree.Close()

	rootNode := tree.RootNode()
	serializers := extractSerializers(rootNode, source)

	ser, ok := serializers["ConfigSerializer"]
	if !ok {
		t.Fatal("expected 'ConfigSerializer'")
	}

	nameField := findSerializerField(ser.Fields, "name")
	if nameField == nil {
		t.Fatal("expected 'name' field")
	}
	if nameField.Required {
		t.Error("name.Required = true, want false (has default)")
	}
}

func TestExtractSerializerClassFromView(t *testing.T) {
	source := []byte(`
from rest_framework import viewsets

class UserViewSet(viewsets.ModelViewSet):
    serializer_class = UserSerializer

    def list(self, request):
        pass
`)

	p := createTestParser(t)
	tree := p.Parse(source, nil)
	defer tree.Close()

	rootNode := tree.RootNode()

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child.Kind() == "class_definition" {
			className := extractClassName(child, source)
			if className == "UserViewSet" {
				serClass := extractSerializerClassFromView(child, source)
				if serClass != "UserSerializer" {
					t.Errorf("serializer_class = %q, want %q", serClass, "UserSerializer")
				}
				return
			}
		}
	}
	t.Fatal("UserViewSet class not found")
}

func TestDRFFieldTypeMap(t *testing.T) {
	tests := []struct {
		drfType  string
		jsonType string
	}{
		{"CharField", "string"},
		{"TextField", "string"},
		{"EmailField", "string"},
		{"IntegerField", "int"},
		{"BigIntegerField", "long"},
		{"FloatField", "float"},
		{"DecimalField", "float"},
		{"BooleanField", "boolean"},
		{"DateField", "string"},
		{"DateTimeField", "string"},
		{"ListField", "array"},
		{"DictField", "map"},
		{"SerializerMethodField", "string"},
	}

	for _, tt := range tests {
		jsonType, ok := drfFieldTypeMap[tt.drfType]
		if !ok {
			t.Errorf("DRF type %q not found in mapping", tt.drfType)
			continue
		}
		if jsonType != tt.jsonType {
			t.Errorf("drfFieldTypeMap[%q] = %q, want %q", tt.drfType, jsonType, tt.jsonType)
		}
	}
}

func findSerializerField(fields []SerializerField, name string) *SerializerField {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

func createTestParser(t *testing.T) *tree_sitter.Parser {
	t.Helper()
	p := tree_sitter.NewParser()
	lang := tree_sitter.NewLanguage(python.Language())
	if err := p.SetLanguage(lang); err != nil {
		t.Fatalf("failed to set language: %v", err)
	}
	return p
}

func TestDRFTypeResolver_PrimitiveFields(t *testing.T) {
	serializers := map[string]SerializerModel{
		"UserSerializer": {
			Name: "UserSerializer",
			Fields: []SerializerField{
				{Name: "name", DRFType: "CharField", Required: true},
				{Name: "email", DRFType: "EmailField", Required: true},
				{Name: "age", DRFType: "IntegerField", Required: false},
			},
		},
	}

	resolver := NewDRFTypeResolver(serializers)

	result := resolver.ResolveSerializer("UserSerializer")
	if result.Kind != model.KindObject {
		t.Fatalf("ResolveSerializer(UserSerializer).Kind = %q, want %q", result.Kind, model.KindObject)
	}
	if result.TypeName != "UserSerializer" {
		t.Errorf("ResolveSerializer(UserSerializer).TypeName = %q, want %q", result.TypeName, "UserSerializer")
	}
	if len(result.Fields) != 3 {
		t.Fatalf("fields count = %d, want 3", len(result.Fields))
	}

	nameField, ok := result.Fields["name"]
	if !ok {
		t.Fatal("expected 'name' field")
	}
	if nameField.Model.TypeName != model.JsonTypeString {
		t.Errorf("name model type = %q, want %q", nameField.Model.TypeName, model.JsonTypeString)
	}

	ageField, ok := result.Fields["age"]
	if !ok {
		t.Fatal("expected 'age' field")
	}
	if ageField.Model.TypeName != model.JsonTypeInt {
		t.Errorf("age model type = %q, want %q", ageField.Model.TypeName, model.JsonTypeInt)
	}
}

func TestDRFTypeResolver_NestedSerializer(t *testing.T) {
	serializers := map[string]SerializerModel{
		"AddressSerializer": {
			Name: "AddressSerializer",
			Fields: []SerializerField{
				{Name: "street", DRFType: "CharField", Required: true},
				{Name: "city", DRFType: "CharField", Required: true},
			},
		},
		"UserWithAddressSerializer": {
			Name: "UserWithAddressSerializer",
			Fields: []SerializerField{
				{Name: "name", DRFType: "CharField", Required: true},
				{Name: "address", DRFType: "AddressSerializer", Required: true},
			},
		},
	}

	resolver := NewDRFTypeResolver(serializers)

	result := resolver.ResolveSerializer("UserWithAddressSerializer")
	if result.Kind != model.KindObject {
		t.Fatalf("Kind = %q, want %q", result.Kind, model.KindObject)
	}

	addrField, ok := result.Fields["address"]
	if !ok {
		t.Fatal("expected 'address' field")
	}
	if addrField.Model.Kind != model.KindObject {
		t.Errorf("address kind = %q, want %q", addrField.Model.Kind, model.KindObject)
	}
	if addrField.Model.TypeName != "AddressSerializer" {
		t.Errorf("address typeName = %q, want %q", addrField.Model.TypeName, "AddressSerializer")
	}
	if len(addrField.Model.Fields) != 2 {
		t.Errorf("address fields count = %d, want 2", len(addrField.Model.Fields))
	}
}

func TestDRFTypeResolver_Inheritance(t *testing.T) {
	serializers := map[string]SerializerModel{
		"BaseSerializer": {
			Name: "BaseSerializer",
			Fields: []SerializerField{
				{Name: "id", DRFType: "IntegerField", Required: true, ReadOnly: true},
				{Name: "created_at", DRFType: "DateTimeField", Required: true, ReadOnly: true},
			},
		},
		"ItemSerializer": {
			Name: "ItemSerializer",
			Fields: []SerializerField{
				{Name: "name", DRFType: "CharField", Required: true},
				{Name: "price", DRFType: "FloatField", Required: true},
			},
			EmbeddedTypes: []string{"BaseSerializer"},
		},
	}

	resolver := NewDRFTypeResolver(serializers)

	result := resolver.ResolveSerializer("ItemSerializer")
	if result.Kind != model.KindObject {
		t.Fatalf("Kind = %q, want %q", result.Kind, model.KindObject)
	}

	if len(result.Fields) != 4 {
		t.Fatalf("fields count = %d, want 4 (2 own + 2 inherited)", len(result.Fields))
	}

	if _, ok := result.Fields["id"]; !ok {
		t.Error("expected inherited 'id' field")
	}
	if _, ok := result.Fields["created_at"]; !ok {
		t.Error("expected inherited 'created_at' field")
	}
	if _, ok := result.Fields["name"]; !ok {
		t.Error("expected own 'name' field")
	}
	if _, ok := result.Fields["price"]; !ok {
		t.Error("expected own 'price' field")
	}
}

func TestDRFTypeResolver_ReadOnlyFields(t *testing.T) {
	serializers := map[string]SerializerModel{
		"ReadOnlySerializer": {
			Name: "ReadOnlySerializer",
			Fields: []SerializerField{
				{Name: "id", DRFType: "IntegerField", Required: true, ReadOnly: true},
				{Name: "name", DRFType: "CharField", Required: true},
			},
		},
	}

	resolver := NewDRFTypeResolver(serializers)

	result := resolver.ResolveSerializer("ReadOnlySerializer")

	idField, ok := result.Fields["id"]
	if !ok {
		t.Fatal("expected 'id' field")
	}
	if idField.Required {
		t.Error("id.Required = true for read_only field, want false")
	}

	nameField, ok := result.Fields["name"]
	if !ok {
		t.Fatal("expected 'name' field")
	}
	if !nameField.Required {
		t.Error("name.Required = false, want true")
	}
}

func TestDRFTypeResolver_ManyField(t *testing.T) {
	serializers := map[string]SerializerModel{
		"TagSerializer": {
			Name: "TagSerializer",
			Fields: []SerializerField{
				{Name: "name", DRFType: "CharField", Required: true},
			},
		},
		"ArticleSerializer": {
			Name: "ArticleSerializer",
			Fields: []SerializerField{
				{Name: "title", DRFType: "CharField", Required: true},
				{Name: "tags", DRFType: "TagSerializer", Required: true, Many: true},
			},
		},
	}

	resolver := NewDRFTypeResolver(serializers)

	result := resolver.ResolveSerializer("ArticleSerializer")

	tagsField, ok := result.Fields["tags"]
	if !ok {
		t.Fatal("expected 'tags' field")
	}
	if tagsField.Model.Kind != model.KindArray {
		t.Errorf("tags kind = %q, want %q", tagsField.Model.Kind, model.KindArray)
	}
	if tagsField.Model.Items == nil {
		t.Fatal("expected Items to be non-nil")
	}
	if tagsField.Model.Items.Kind != model.KindObject {
		t.Errorf("tags items kind = %q, want %q", tagsField.Model.Items.Kind, model.KindObject)
	}
}

func TestDRFTypeResolver_CircularReference(t *testing.T) {
	serializers := map[string]SerializerModel{
		"NodeSerializer": {
			Name: "NodeSerializer",
			Fields: []SerializerField{
				{Name: "value", DRFType: "CharField", Required: true},
				{Name: "child", DRFType: "NodeSerializer", Required: false},
			},
		},
	}

	resolver := NewDRFTypeResolver(serializers)

	result := resolver.ResolveSerializer("NodeSerializer")
	if result.Kind != model.KindObject {
		t.Fatalf("Kind = %q, want %q", result.Kind, model.KindObject)
	}

	childField, ok := result.Fields["child"]
	if !ok {
		t.Fatal("expected 'child' field")
	}
	if childField.Model.Kind != model.KindRef {
		t.Errorf("child kind = %q, want %q (circular ref)", childField.Model.Kind, model.KindRef)
	}
}

func TestDRFTypeResolver_UnknownSerializer(t *testing.T) {
	resolver := NewDRFTypeResolver(nil)

	result := resolver.ResolveSerializer("UnknownSerializer")
	if result.Kind != model.KindSingle {
		t.Errorf("Kind = %q, want %q", result.Kind, model.KindSingle)
	}
	if result.TypeName != "UnknownSerializer" {
		t.Errorf("TypeName = %q, want %q", result.TypeName, "UnknownSerializer")
	}
}

func TestDRFTypeResolver_ActionSerializer(t *testing.T) {
	resolver := NewDRFTypeResolver(nil)

	tests := []struct {
		action    string
		wantReq   bool
		wantResp  bool
	}{
		{"list", false, true},
		{"create", true, true},
		{"retrieve", false, true},
		{"update", true, true},
		{"partial_update", true, true},
		{"destroy", false, false},
	}

	for _, tt := range tests {
		reqSer, respSer := resolver.ResolveActionSerializer("TestViewSet", tt.action, "TestSerializer")
		if tt.wantReq && reqSer == "" {
			t.Errorf("action %q: expected request serializer, got empty", tt.action)
		}
		if !tt.wantReq && reqSer != "" {
			t.Errorf("action %q: expected no request serializer, got %q", tt.action, reqSer)
		}
		if tt.wantResp && respSer == "" {
			t.Errorf("action %q: expected response serializer, got empty", tt.action)
		}
		if !tt.wantResp && respSer != "" {
			t.Errorf("action %q: expected no response serializer, got %q", tt.action, respSer)
		}
	}
}

func TestDRFTypeResolver_HTTPMethodSerializer(t *testing.T) {
	resolver := NewDRFTypeResolver(nil)

	tests := []struct {
		method   string
		wantReq  bool
		wantResp bool
	}{
		{"GET", false, true},
		{"POST", true, true},
		{"PUT", true, true},
		{"PATCH", true, true},
		{"DELETE", false, false},
	}

	for _, tt := range tests {
		reqSer, respSer := resolver.ResolveHTTPMethodSerializer("TestView", tt.method, "TestSerializer")
		if tt.wantReq && reqSer == "" {
			t.Errorf("method %q: expected request serializer, got empty", tt.method)
		}
		if !tt.wantReq && reqSer != "" {
			t.Errorf("method %q: expected no request serializer, got %q", tt.method, reqSer)
		}
		if tt.wantResp && respSer == "" {
			t.Errorf("method %q: expected response serializer, got empty", tt.method)
		}
		if !tt.wantResp && respSer != "" {
			t.Errorf("method %q: expected no response serializer, got %q", tt.method, respSer)
		}
	}
}

func TestDRFTypeResolver_BuildRequestBody(t *testing.T) {
	serializers := map[string]SerializerModel{
		"UserSerializer": {
			Name: "UserSerializer",
			Fields: []SerializerField{
				{Name: "name", DRFType: "CharField", Required: true},
				{Name: "email", DRFType: "EmailField", Required: true},
			},
		},
	}

	resolver := NewDRFTypeResolver(serializers)

	body := resolver.BuildRequestBody("UserSerializer")
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	if body.Kind != model.KindObject {
		t.Errorf("body.Kind = %q, want %q", body.Kind, model.KindObject)
	}
	if body.TypeName != "UserSerializer" {
		t.Errorf("body.TypeName = %q, want %q", body.TypeName, "UserSerializer")
	}
}

func TestDRFTypeResolver_BuildResponseBody(t *testing.T) {
	serializers := map[string]SerializerModel{
		"UserSerializer": {
			Name: "UserSerializer",
			Fields: []SerializerField{
				{Name: "name", DRFType: "CharField", Required: true},
				{Name: "email", DRFType: "EmailField", Required: true},
			},
		},
	}

	resolver := NewDRFTypeResolver(serializers)

	body := resolver.BuildResponseBody("UserSerializer")
	if body == nil {
		t.Fatal("expected non-nil body")
	}
	if body.Kind != model.KindObject {
		t.Errorf("body.Kind = %q, want %q", body.Kind, model.KindObject)
	}
}

func TestDRFTypeResolver_BuildBody_Empty(t *testing.T) {
	resolver := NewDRFTypeResolver(nil)

	body := resolver.BuildRequestBody("")
	if body != nil {
		t.Error("expected nil body for empty serializer name")
	}

	body = resolver.BuildResponseBody("")
	if body != nil {
		t.Error("expected nil body for empty serializer name")
	}
}

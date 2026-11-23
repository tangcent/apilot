package resolver

import (
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	model "github.com/tangcent/apilot/api-model"
)

func testProperty(name, typ string, annotations ...ktparser.Annotation) ktparser.Property {
	return ktparser.Property{Name: name, Type: typ, Annotations: annotations}
}

func testAnnotation(name string, params map[string]string) ktparser.Annotation {
	return ktparser.Annotation{Name: name, Params: params}
}

func TestNormalizeType(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "int", raw: "Int", want: "java.lang.Integer"},
		{name: "byte", raw: "Byte", want: "java.lang.Integer"},
		{name: "short", raw: "Short", want: "java.lang.Integer"},
		{name: "unsigned int", raw: "UInt", want: "java.lang.Integer"},
		{name: "unsigned short", raw: "UShort", want: "java.lang.Integer"},
		{name: "long", raw: "Long", want: "java.lang.Long"},
		{name: "unsigned long", raw: "ULong", want: "java.lang.Long"},
		{name: "float", raw: "Float", want: "java.lang.Float"},
		{name: "double", raw: "Double", want: "java.lang.Double"},
		{name: "boolean", raw: "Boolean", want: "java.lang.Boolean"},
		{name: "char becomes a string", raw: "Char", want: "java.lang.String"},
		{name: "string", raw: "String", want: "java.lang.String"},
		{name: "any becomes object", raw: "Any", want: "java.lang.Object"},
		{name: "unit becomes void", raw: "Unit", want: "java.lang.Void"},
		{name: "nothing becomes void", raw: "Nothing", want: "java.lang.Void"},
		{name: "number", raw: "Number", want: "java.lang.Number"},
		{name: "java.lang integer", raw: "Integer", want: "java.lang.Integer"},
		{name: "java.lang object", raw: "Object", want: "java.lang.Object"},
		{name: "java.lang character", raw: "Character", want: "java.lang.Character"},
		{name: "qualified int", raw: "kotlin.Int", want: "java.lang.Integer"},
		{name: "qualified java.lang", raw: "java.lang.Integer", want: "java.lang.Integer"},
		{name: "star projection", raw: "*", want: "?"},
		{name: "empty", raw: "  ", want: ""},
		{name: "simple class unchanged", raw: "User", want: "User"},
		{name: "enum unchanged", raw: "Status", want: "Status"},
		{name: "qualified class unchanged", raw: "com.example.Number", want: "com.example.Number"},
		{name: "list", raw: "List<Int>", want: "java.util.List<java.lang.Integer>"},
		{name: "qualified list", raw: "kotlin.collections.List<kotlin.String>", want: "java.util.List<java.lang.String>"},
		{name: "java list", raw: "java.util.List<String>", want: "java.util.List<java.lang.String>"},
		{name: "mutable list", raw: "MutableList<User>", want: "java.util.List<User>"},
		{name: "iterable", raw: "Iterable<User>", want: "java.util.List<User>"},
		{name: "sequence", raw: "Sequence<Int>", want: "java.util.List<java.lang.Integer>"},
		{name: "array", raw: "Array<String>", want: "java.util.List<java.lang.String>"},
		{name: "set", raw: "Set<Long>", want: "java.util.Set<java.lang.Long>"},
		{name: "hash set", raw: "HashSet<Int>", want: "java.util.HashSet<java.lang.Integer>"},
		{name: "mutable set", raw: "MutableSet<Int>", want: "java.util.Set<java.lang.Integer>"},
		{name: "map", raw: "Map<String, Int>", want: "java.util.Map<java.lang.String,java.lang.Integer>"},
		{name: "hash map nests", raw: "HashMap<String, List<Int>>", want: "java.util.Map<java.lang.String,java.util.List<java.lang.Integer>>"},
		{name: "mutable map", raw: "MutableMap<String, Boolean>", want: "java.util.Map<java.lang.String,java.lang.Boolean>"},
		{name: "int array", raw: "IntArray", want: "java.util.List<java.lang.Integer>"},
		{name: "byte array", raw: "ByteArray", want: "java.util.List<java.lang.Integer>"},
		{name: "long array", raw: "LongArray", want: "java.util.List<java.lang.Long>"},
		{name: "boolean array", raw: "BooleanArray", want: "java.util.List<java.lang.Boolean>"},
		{name: "char array", raw: "CharArray", want: "java.util.List<java.lang.String>"},
		{name: "double array", raw: "DoubleArray", want: "java.util.List<java.lang.Double>"},
		{name: "nested null stripped upstream already", raw: "List<Any>", want: "java.util.List<java.lang.Object>"},
		{name: "star inside a list", raw: "List<*>", want: "java.util.List<?>"},
		{name: "response entity", raw: "ResponseEntity<Result<User>>", want: "ResponseEntity<Result<User>>"},
		{name: "a name no scope binds is as written", raw: "LinkedList<Int>", want: "LinkedList<java.lang.Integer>"},
	}

	r := New(nil)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := r.NormalizeType(&ktparser.Class{Name: "Source", Package: "com.example.app"}, tt.raw); got != tt.want {
				t.Errorf("NormalizeType(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestFindClass(t *testing.T) {
	modelPkg := "com.example.model"
	sameFile := ktparser.Class{Name: "Helper", Package: modelPkg, FilePath: "A.kt"}
	imported := ktparser.Class{Name: "Helper", Package: "com.example.other", FilePath: "B.kt"}
	unique := ktparser.Class{Name: "Unique", Package: "com.example.unique", FilePath: "C.kt"}
	classes := []ktparser.Class{sameFile, imported, unique}

	r := New(classes)

	t.Run("same file wins", func(t *testing.T) {
		source := ktparser.Class{Name: "Source", Package: modelPkg, FilePath: "A.kt"}
		got, ok := r.FindClass(&source, "Helper")
		if !ok || got.FilePath != "A.kt" {
			t.Fatalf("got %#v, ok=%v", got, ok)
		}
	})

	t.Run("explicit import beats same package", func(t *testing.T) {
		source := ktparser.Class{
			Name:     "Source",
			Package:  modelPkg,
			FilePath: "D.kt",
			Imports:  []ktparser.Import{{Path: "com.example.other.Helper"}},
		}
		got, ok := r.FindClass(&source, "Helper")
		if !ok || got.Package != "com.example.other" {
			t.Fatalf("got %#v, ok=%v", got, ok)
		}
	})

	t.Run("same package", func(t *testing.T) {
		source := ktparser.Class{Name: "Source2", Package: modelPkg, FilePath: "D.kt"}
		got, ok := r.FindClass(&source, "Helper")
		if !ok || got.Package != modelPkg {
			t.Fatalf("got %#v, ok=%v", got, ok)
		}
	})

	t.Run("wildcard import", func(t *testing.T) {
		source := ktparser.Class{
			Name:     "Source",
			Package:  "com.example.web",
			FilePath: "E.kt",
			Imports:  []ktparser.Import{{Path: "com.example.model", OnDemand: true}},
		}
		got, ok := r.FindClass(&source, "Helper")
		if !ok || got.Package != modelPkg {
			t.Fatalf("got %#v, ok=%v", got, ok)
		}
	})

	t.Run("fully qualified", func(t *testing.T) {
		source := ktparser.Class{Name: "Source", Package: "com.example.web", FilePath: "E.kt"}
		got, ok := r.FindClass(&source, "com.example.other.Helper")
		if !ok || got.Package != "com.example.other" {
			t.Fatalf("got %#v, ok=%v", got, ok)
		}
	})

	t.Run("unique simple name", func(t *testing.T) {
		source := ktparser.Class{Name: "Source", Package: "com.example.web", FilePath: "E.kt"}
		got, ok := r.FindClass(&source, "Unique")
		if !ok || got.Name != "Unique" {
			t.Fatalf("got %#v, ok=%v", got, ok)
		}
	})

	t.Run("unknown name", func(t *testing.T) {
		source := ktparser.Class{Name: "Source", Package: "com.example.web", FilePath: "E.kt"}
		if got, ok := r.FindClass(&source, "Missing"); ok {
			t.Fatalf("unexpected class: %#v", got)
		}
	})
}

func resolveTestClasses() []ktparser.Class {
	return []ktparser.Class{
		{
			Name:           "Result",
			Package:        "com.example.model",
			FilePath:       "Result.kt",
			TypeParameters: []string{"T"},
			Fields: []ktparser.Property{
				testProperty("code", "Int"),
				testProperty("message", "String"),
				testProperty("data", "T"),
			},
		},
		{
			Name:           "PageResult",
			Package:        "com.example.model",
			FilePath:       "PageResult.kt",
			TypeParameters: []string{"T"},
			Fields: []ktparser.Property{
				testProperty("total", "Long"),
				testProperty("items", "List<T>"),
			},
		},
		{
			Name:     "User",
			Package:  "com.example.model",
			FilePath: "User.kt",
			Fields: []ktparser.Property{
				testProperty("id", "Long"),
				testProperty("name", "String"),
			},
		},
		{
			Name:       "Status",
			Package:    "com.example.model",
			FilePath:   "Status.kt",
			IsEnum:     true,
			EnumValues: []string{"ACTIVE", "DISABLED"},
		},
		{
			Name:     "Category",
			Package:  "com.example.model",
			FilePath: "Category.kt",
			Fields: []ktparser.Property{
				testProperty("status", "Status"),
				testProperty("owner", "User"),
				testProperty("children", "List<Category>"),
			},
		},
	}
}

func TestResolveFromKotlinTypes(t *testing.T) {
	source := ktparser.Class{
		Name:     "Source",
		Package:  "com.example",
		FilePath: "Source.kt",
		Imports: []ktparser.Import{
			{Path: "com.example.model.Result"},
			{Path: "com.example.model.PageResult"},
			{Path: "com.example.model.User"},
			{Path: "com.example.model.Category"},
		},
	}
	r := New(append(resolveTestClasses(), source))

	t.Run("nested generic substitutes through both levels", func(t *testing.T) {
		got := r.ResolveFrom(&source, "Result<PageResult<User>>", nil)
		if got == nil || got.Kind != model.KindObject || got.TypeName != "Result" {
			t.Fatalf("result model = %#v", got)
		}
		data := objectField(t, got, "data")
		if data.Model.Kind != model.KindObject || data.Model.TypeName != "PageResult" {
			t.Fatalf("data model = %#v", data.Model)
		}
		items := objectField(t, data.Model, "items")
		if items.Model.Kind != model.KindArray || items.Model.Items.TypeName != "User" {
			t.Fatalf("items model = %#v", items.Model)
		}
		if _, ok := items.Model.Items.Fields["name"]; !ok {
			t.Fatalf("User fields missing: %v", items.Model.Items.Fields)
		}
	})

	t.Run("Any becomes an empty object", func(t *testing.T) {
		got := r.ResolveFrom(&source, "Result<PageResult<Any>>", nil)
		items := objectField(t, objectField(t, got, "data").Model, "items")
		if items.Model.Kind != model.KindArray || items.Model.Items.Kind != model.KindObject {
			t.Fatalf("items model = %#v", items.Model)
		}
	})

	t.Run("Void becomes null", func(t *testing.T) {
		got := r.ResolveFrom(&source, "Result<Void>", nil)
		data := objectField(t, got, "data")
		if data.Model.Kind != model.KindSingle || data.Model.TypeName != model.JsonTypeNull {
			t.Fatalf("data model = %#v", data.Model)
		}
	})

	t.Run("empty type is null", func(t *testing.T) {
		if got := r.ResolveFrom(&source, "", nil); !got.IsNull() {
			t.Fatalf("got %#v", got)
		}
	})

	t.Run("recursive type terminates with a reference", func(t *testing.T) {
		got := r.ResolveFrom(&source, "Category", nil)
		children := objectField(t, got, "children")
		if children.Model.Kind != model.KindArray || !children.Model.Items.IsRef() {
			t.Fatalf("children model = %#v", children.Model)
		}
		if children.Model.Items.TypeName != "Category" {
			t.Fatalf("recursive reference = %q", children.Model.Items.TypeName)
		}
	})

	t.Run("enum field becomes a string with options", func(t *testing.T) {
		got := r.ResolveFrom(&source, "Category", nil)
		status := objectField(t, got, "status")
		if status.Model.TypeName != model.JsonTypeString {
			t.Fatalf("status model = %#v", status.Model)
		}
		if len(status.Options) != 2 || status.Options[0].Value != "ACTIVE" || status.Options[1].Value != "DISABLED" {
			t.Fatalf("status options = %#v", status.Options)
		}
	})

	t.Run("enum is not reported unresolved", func(t *testing.T) {
		r.ResolveFrom(&source, "Category", nil)
		if _, ok := r.Unresolved()["Status"]; ok {
			t.Fatalf("Status reported unresolved: %v", r.Unresolved())
		}
		if values := r.EnumValues("com.example.model.Status"); len(values) != 2 {
			t.Fatalf("EnumValues = %v", values)
		}
	})
}

func TestToJavaClassFieldMapping(t *testing.T) {
	class := ktparser.Class{
		Name:    "CreateReq",
		Package: "com.example.model",
		Fields: []ktparser.Property{
			testProperty("secret", "String", testAnnotation("JsonIgnore", nil)),
			testProperty("revived", "String",
				testAnnotation("JsonIgnore", map[string]string{"value": "false"}),
				testAnnotation("JsonProperty", map[string]string{"value": "revived_name"})),
			testProperty("displayName", "String", testAnnotation("JsonProperty", map[string]string{"value": "display_name"})),
			testProperty("parentId", "Long", testAnnotation("JsonProperty", map[string]string{"required": "false"})),
			testProperty("name", "String"),
		},
		Methods: []ktparser.Function{{Name: "getDisplayName"}},
	}

	got := newTypeScope([]ktparser.Class{class}, nil).toJavaClass(class)
	if len(got.Methods) != 0 {
		t.Errorf("compiler-generated methods leaked: %#v", got.Methods)
	}

	names := map[string]bool{}
	for _, field := range got.Fields {
		names[field.Name] = true
		if field.Name == "parentId" {
			found := false
			for _, ann := range field.Annotations {
				if ann.Name == "Nullable" {
					found = true
				}
			}
			if !found {
				t.Errorf("parentId annotations = %#v, want Nullable", field.Annotations)
			}
		}
	}
	if names["secret"] {
		t.Error("JsonIgnore field was copied")
	}
	if !names["revived_name"] || names["revived"] {
		t.Errorf("@JsonIgnore(false) should not drop the field and the rename should apply: %v", names)
	}
	if !names["display_name"] || names["displayName"] {
		t.Errorf("JsonProperty rename not applied: %v", names)
	}
	if !names["name"] {
		t.Errorf("plain field missing: %v", names)
	}
}

func objectField(t *testing.T, object *model.ObjectModel, name string) *model.FieldModel {
	t.Helper()
	if object == nil || object.Kind != model.KindObject {
		t.Fatalf("expected object model, got %#v", object)
	}
	field, ok := object.Fields[name]
	if !ok {
		t.Fatalf("field %q not found in %v", name, fieldNames(object))
	}
	return field
}

func fieldNames(object *model.ObjectModel) []string {
	names := make([]string, 0, len(object.Fields))
	for name := range object.Fields {
		names = append(names, name)
	}
	return names
}

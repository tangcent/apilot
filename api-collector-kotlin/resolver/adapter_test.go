package resolver

import (
	"reflect"
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	model "github.com/tangcent/apilot/api-model"
)

func testProperty(name, typ string, annotations ...ktparser.Annotation) ktparser.Property {
	return ktparser.Property{Name: name, Type: typ, Annotations: annotations}
}

// mutableProperty is a var, which Spring can bind from a request parameter.
func mutableProperty(name, typ string) ktparser.Property {
	return ktparser.Property{Name: name, Type: typ, IsMutable: true}
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
		{name: "byte array is a java byte array", raw: "ByteArray", want: "byte[]"},
		{name: "qualified byte array", raw: "kotlin.ByteArray", want: "byte[]"},
		{name: "list of byte arrays", raw: "List<ByteArray>", want: "java.util.List<byte[]>"},
		{name: "array of byte arrays", raw: "Array<ByteArray>", want: "java.util.List<byte[]>"},
		{name: "array of boxed bytes stays numbers", raw: "Array<Byte>", want: "java.util.List<java.lang.Integer>"},
		{name: "list of bytes stays numbers", raw: "List<Byte>", want: "java.util.List<java.lang.Integer>"},
		{name: "long array", raw: "LongArray", want: "java.util.List<java.lang.Long>"},
		{name: "boolean array", raw: "BooleanArray", want: "java.util.List<java.lang.Boolean>"},
		{name: "char array is a java char array", raw: "CharArray", want: "char[]"},
		{name: "qualified char array", raw: "kotlin.CharArray", want: "char[]"},
		{name: "list of char arrays", raw: "List<CharArray>", want: "java.util.List<char[]>"},
		{name: "array of char arrays", raw: "Array<CharArray>", want: "java.util.List<char[]>"},
		{name: "array of boxed chars stays strings", raw: "Array<Char>", want: "java.util.List<java.lang.String>"},
		{name: "list of chars stays strings", raw: "List<Char>", want: "java.util.List<java.lang.String>"},
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
		if values, ok := r.EnumValuesFrom(&source, "Status"); !ok || len(values) != 2 {
			t.Fatalf("EnumValuesFrom = %v, %v", values, ok)
		}
	})
}

func TestToJavaEnumJsonValue(t *testing.T) {
	jsonValue := func(target string, params map[string]string) ktparser.Annotation {
		return ktparser.Annotation{Name: "JsonValue", Params: params, UseSiteTarget: target}
	}
	tests := []struct {
		name  string
		class ktparser.Class
		want  bool
	}{
		{name: "no annotation", class: ktparser.Class{
			Fields: []ktparser.Property{testProperty("code", "String")},
		}},
		{name: "on a constructor property", class: ktparser.Class{
			Fields: []ktparser.Property{testProperty("code", "String", jsonValue("", nil))},
		}, want: true},
		{name: "on the getter", class: ktparser.Class{
			Fields: []ktparser.Property{testProperty("code", "String", jsonValue("get", nil))},
		}, want: true},
		{name: "on the field", class: ktparser.Class{
			Fields: []ktparser.Property{testProperty("code", "String", jsonValue("field", nil))},
		}, want: true},
		{name: "on the constructor parameter, where Jackson does not read it as the value", class: ktparser.Class{
			Fields: []ktparser.Property{testProperty("code", "String", jsonValue("param", nil))},
		}},
		{name: "on the setter", class: ktparser.Class{
			Fields: []ktparser.Property{testProperty("code", "String", jsonValue("set", nil))},
		}},
		{name: "switched off", class: ktparser.Class{
			Fields: []ktparser.Property{testProperty("code", "String", jsonValue("", map[string]string{"value": "false"}))},
		}},
		{name: "on a function", class: ktparser.Class{
			Methods: []ktparser.Function{{Name: "toValue", Annotations: []ktparser.Annotation{jsonValue("", nil)}}},
		}, want: true},
		{name: "another annotation", class: ktparser.Class{
			Fields: []ktparser.Property{testProperty("code", "String", testAnnotation("JsonProperty", nil))},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.class.IsEnum = true
			if got := toJavaEnum(tt.class).EnumJsonValue; got != tt.want {
				t.Errorf("EnumJsonValue = %v, want %v", got, tt.want)
			}
		})
	}

	shaped := toJavaEnum(ktparser.Class{
		IsEnum:      true,
		Annotations: []ktparser.Annotation{testAnnotation("JsonFormat", map[string]string{"shape": "JsonFormat.Shape.OBJECT"})},
	})
	if len(shaped.Annotations) != 1 || shaped.Annotations[0].Params["shape"] != "JsonFormat.Shape.OBJECT" {
		t.Errorf("the annotations of the enum are not carried over: %+v", shaped.Annotations)
	}
}

// A property typed with an enum is a string of its entries, as in a Java
// declaration of the same shape; an enum whose JSON form is not the name of an
// entry keeps its type name and is reported unresolved.
func TestResolveFromEnumsByTheirJSONForm(t *testing.T) {
	classes := []ktparser.Class{
		{Name: "Level", Package: "com.example", FilePath: "Level.kt", IsEnum: true, EnumValues: []string{"LOW", "HIGH"}},
		{
			Name: "Code", Package: "com.example", FilePath: "Code.kt", IsEnum: true, EnumValues: []string{"X", "Y"},
			Fields: []ktparser.Property{testProperty("code", "String", testAnnotation("JsonValue", nil))},
		},
		{
			Name: "Shape", Package: "com.example", FilePath: "Shape.kt", IsEnum: true, EnumValues: []string{"CIRCLE"},
			Annotations: []ktparser.Annotation{testAnnotation("JsonFormat", map[string]string{"shape": "JsonFormat.Shape.OBJECT"})},
		},
		{
			Name: "Form", Package: "com.example", FilePath: "Form.kt",
			Fields: []ktparser.Property{
				testProperty("level", "Level"),
				testProperty("levels", "List<Level>"),
				testProperty("byName", "Map<String, Level>"),
				testProperty("code", "Code"),
				testProperty("shape", "Shape"),
			},
		},
	}
	r := New(classes)
	source := classes[len(classes)-1]
	form := r.ResolveFrom(&source, "Form", nil)

	wantLevels := []string{"LOW", "HIGH"}
	for _, name := range []string{"level", "levels"} {
		field := objectField(t, form, name)
		var got []string
		for _, option := range field.Options {
			got = append(got, option.Value.(string))
		}
		if !reflect.DeepEqual(got, wantLevels) {
			t.Errorf("%s options = %v, want %v", name, got, wantLevels)
		}
	}
	if levels := objectField(t, form, "levels").Model; levels.Kind != model.KindArray || levels.Items.TypeName != model.JsonTypeString {
		t.Errorf("levels = %+v, want an array of strings", levels)
	}
	// The values of a map are strings of the entries, and the map lists none.
	if byName := objectField(t, form, "byName"); !byName.Model.IsMap() || byName.Model.ValueModel.TypeName != model.JsonTypeString || len(byName.Options) != 0 {
		t.Errorf("byName = %+v, options %v, want a map of strings without options", byName.Model, byName.Options)
	}
	for _, name := range []string{"code", "shape"} {
		field := objectField(t, form, name)
		if want := map[string]string{"code": "Code", "shape": "Shape"}[name]; field.Model.TypeName != want || len(field.Options) != 0 {
			t.Errorf("%s = %+v, options %v, want the type name %s", name, field.Model, field.Options, want)
		}
	}
	if got := r.Unresolved(); !reflect.DeepEqual(got, map[string]int{"Code": 1, "Shape": 1}) {
		t.Errorf("unresolved = %v, want the two enums with another JSON form", got)
	}
}

// An enum nested in a class is named through the class, Order.Status, or by an
// import of that path: it is a string of its entries all the same.
func TestResolveFromNestedEnum(t *testing.T) {
	classes := []ktparser.Class{
		{Name: "Order", Package: "com.example", FilePath: "Order.kt"},
		{Name: "Status", Package: "com.example", Outer: "Order", FilePath: "Order.kt", IsEnum: true, EnumValues: []string{"OPEN", "CLOSED"}},
		{
			Name: "Holder", Package: "com.other", FilePath: "Holder.kt",
			Imports: []ktparser.Import{{Path: "com.example.Order.Status"}},
			Fields: []ktparser.Property{
				testProperty("imported", "Status"),
				testProperty("qualified", "com.example.Order.Status"),
				testProperty("list", "List<com.example.Order.Status>"),
			},
		},
	}
	r := New(classes)
	source := classes[len(classes)-1]
	holder := r.ResolveFrom(&source, "Holder", nil)

	for _, name := range []string{"imported", "qualified", "list"} {
		field := objectField(t, holder, name)
		if len(field.Options) != 2 || field.Options[0].Value != "OPEN" || field.Options[1].Value != "CLOSED" {
			t.Errorf("%s = %+v, options %v, want the entries of the enum", name, field.Model, field.Options)
		}
	}
	if got := r.Unresolved(); len(got) != 0 {
		t.Errorf("unresolved = %v, want none", got)
	}
}

// A property names an enum as the scope of its source gives the name: the enum of
// the source's own package, even where the JSON view leaves it unresolved, and the
// enum nested in the Order that scope gives. Another package's enum of the same
// simple name does not stand in for either.
func TestResolveFromEnumsInTheScopeOfTheSource(t *testing.T) {
	properties := []ktparser.Property{
		testProperty("code", "Code"),
		testProperty("status", "Order.Status"),
		testProperty("history", "List<Order.Status>"),
	}
	holderA := ktparser.Class{Name: "Holder", Package: "a", FilePath: "a/Holder.kt", Fields: properties}
	holderB := ktparser.Class{Name: "Holder", Package: "b", FilePath: "b/Holder.kt", Fields: properties}
	classes := []ktparser.Class{
		{
			Name: "Code", Package: "a", FilePath: "a/Code.kt", IsEnum: true, EnumValues: []string{"X"},
			Fields: []ktparser.Property{testProperty("code", "String", testAnnotation("JsonValue", nil))},
		},
		{Name: "Code", Package: "b", FilePath: "b/Code.kt", IsEnum: true, EnumValues: []string{"B"}},
		{Name: "Order", Package: "a", FilePath: "a/Order.kt"},
		{Name: "Status", Package: "a", Outer: "Order", FilePath: "a/Order.kt", IsEnum: true, EnumValues: []string{"A1"}},
		{Name: "Order", Package: "b", FilePath: "b/Order.kt"},
		{Name: "Status", Package: "b", Outer: "Order", FilePath: "b/Order.kt", IsEnum: true, EnumValues: []string{"B1"}},
		holderA,
		holderB,
	}
	r := New(classes)

	values := func(field *model.FieldModel) []string {
		var got []string
		for _, option := range field.Options {
			got = append(got, option.Value.(string))
		}
		return got
	}

	a := r.ResolveFrom(&holderA, "Holder", nil)
	if code := objectField(t, a, "code"); code.Model.TypeName != "Code" || len(code.Options) != 0 {
		t.Errorf("a.Holder.code = %+v, options %v, want the type name of a.Code, not the entries of b.Code", code.Model, values(code))
	}
	for name, want := range map[string][]string{"status": {"A1"}, "history": {"A1"}} {
		if got := values(objectField(t, a, name)); !reflect.DeepEqual(got, want) {
			t.Errorf("a.Holder.%s options = %v, want %v", name, got, want)
		}
	}

	b := r.ResolveFrom(&holderB, "Holder", nil)
	for name, want := range map[string][]string{"code": {"B"}, "status": {"B1"}, "history": {"B1"}} {
		if got := values(objectField(t, b, name)); !reflect.DeepEqual(got, want) {
			t.Errorf("b.Holder.%s options = %v, want %v", name, got, want)
		}
	}
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

// Two classes of one file each nest an enum named Status that declares the entry
// A, and only Order.Status declares @JsonValue. A property typed Status inside a
// class is the enum nested in that class, in the JSON view and in the binding
// view, whichever order the declarations come in; a property typed
// Order.Status, in a class that nests neither, is the enum of Order.
func TestResolveFromEnumsNestedInDifferentClassesOfOneFile(t *testing.T) {
	jsonValue := testProperty("code", "String", testAnnotation("JsonValue", nil))
	order := ktparser.Class{Name: "Order", Package: "p", FilePath: "p/Models.kt", Fields: []ktparser.Property{testProperty("own", "Status")}}
	orderStatus := ktparser.Class{
		Name: "Status", Outer: "Order", Package: "p", FilePath: "p/Models.kt", IsEnum: true,
		EnumValues: []string{"A", "B"}, Fields: []ktparser.Property{jsonValue},
	}
	product := ktparser.Class{Name: "Product", Package: "p", FilePath: "p/Models.kt", Fields: []ktparser.Property{testProperty("own", "Status")}}
	productStatus := ktparser.Class{Name: "Status", Outer: "Product", Package: "p", FilePath: "p/Models.kt", IsEnum: true, EnumValues: []string{"A", "C"}}
	holder := ktparser.Class{
		Name: "Holder", Package: "p", FilePath: "p/Holder.kt",
		Fields: []ktparser.Property{mutableProperty("order", "Order.Status"), mutableProperty("product", "Product.Status")},
	}

	values := func(field *model.FieldModel) []string {
		var got []string
		for _, option := range field.Options {
			got = append(got, option.Value.(string))
		}
		return got
	}
	for name, classes := range map[string][]ktparser.Class{
		"as declared": {order, orderStatus, product, productStatus, holder},
		"reversed":    {holder, productStatus, product, orderStatus, order},
	} {
		r := New(classes)

		own := r.ResolveFrom(&order, "Order", nil).Fields["own"]
		if own.Model.TypeName != "Status" || len(own.Options) != 0 {
			t.Errorf("%s: Order.own = %+v, options %v, want the type name of the enum that declares @JsonValue", name, own.Model, values(own))
		}
		if got := values(objectField(t, r.ResolveFrom(&product, "Product", nil), "own")); !reflect.DeepEqual(got, []string{"A", "C"}) {
			t.Errorf("%s: Product.own options = %v, want A and C", name, got)
		}
		h := r.ResolveFrom(&holder, "Holder", nil)
		if got := h.Fields["order"]; got.Model.TypeName != "Order.Status" || len(got.Options) != 0 {
			t.Errorf("%s: Holder.order = %+v, options %v, want the type name of the enum that declares @JsonValue", name, got.Model, values(got))
		}
		if got := values(h.Fields["product"]); !reflect.DeepEqual(got, []string{"A", "C"}) {
			t.Errorf("%s: Holder.product options = %v, want A and C", name, got)
		}

		// The binding view reads every enum, and the one the name denotes.
		bound := r.ResolveForBinding(&holder, "Holder", nil)
		if got := values(bound.Fields["order"]); !reflect.DeepEqual(got, []string{"A", "B"}) {
			t.Errorf("%s: binding Holder.order options = %v, want A and B", name, got)
		}
		if got := values(bound.Fields["product"]); !reflect.DeepEqual(got, []string{"A", "C"}) {
			t.Errorf("%s: binding Holder.product options = %v, want A and C", name, got)
		}
		if got, ok := r.EnumValuesFrom(&product, "Status"); !ok || !reflect.DeepEqual(got, []string{"A", "C"}) {
			t.Errorf("%s: EnumValuesFrom(Product, Status) = %v, %v, want A and C", name, got, ok)
		}
		if got, ok := r.EnumValuesFrom(&order, "Status"); !ok || !reflect.DeepEqual(got, []string{"A", "B"}) {
			t.Errorf("%s: EnumValuesFrom(Order, Status) = %v, %v, want A and B", name, got, ok)
		}
	}
}

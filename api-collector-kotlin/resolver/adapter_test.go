package resolver

import (
	"reflect"
	"testing"

	javaparser "github.com/tangcent/apilot/api-collector-java/parser"
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
			{Path: "com.example.model.Status"},
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
			if got := newTypeScope(nil, nil).toJavaEnum(tt.class).EnumJsonValue; got != tt.want {
				t.Errorf("EnumJsonValue = %v, want %v", got, tt.want)
			}
		})
	}

	shaped := newTypeScope(nil, nil).toJavaEnum(ktparser.Class{
		IsEnum:      true,
		Annotations: []ktparser.Annotation{testAnnotation("JsonFormat", map[string]string{"shape": "JsonFormat.Shape.OBJECT"})},
	})
	if len(shaped.Annotations) != 1 || shaped.Annotations[0].Params["shape"] != "JsonFormat.Shape.OBJECT" {
		t.Errorf("the annotations of the enum are not carried over: %+v", shaped.Annotations)
	}
}

// toJavaEnum passes the member that carries @JsonValue, under the name the JVM
// gives it, including one switched off. An interface passes the same member, so
// an override can be matched by name. @get: lands on the getter; a property
// whose name starts with is keeps that name. The members are handed over for
// @JsonValue alone: Methods and Fields stay empty, as for any Kotlin class.
func TestToJavaEnumPassesTheMemberThatOverridesJsonValue(t *testing.T) {
	off := ktparser.Annotation{Name: "JsonValue", Params: map[string]string{"value": "false"}}
	enum := newTypeScope(nil, nil).toJavaEnum(ktparser.Class{
		IsEnum: true,
		Methods: []ktparser.Function{{
			Name:        "code",
			Annotations: []ktparser.Annotation{off},
		}},
		Fields: []ktparser.Property{
			testProperty("wire", "String", ktparser.Annotation{Name: "JsonValue", UseSiteTarget: "get"}),
			testProperty("isActive", "Boolean", ktparser.Annotation{Name: "JsonValue", Params: map[string]string{"value": "false"}, UseSiteTarget: "get"}),
			testProperty("code", "String", ktparser.Annotation{Name: "JsonValue", UseSiteTarget: "field"}),
		},
	})
	if len(enum.Methods) != 0 || len(enum.Fields) != 0 {
		t.Errorf("enum methods = %+v, fields = %+v, want none", enum.Methods, enum.Fields)
	}
	got := map[string]string{}
	for _, m := range enum.JsonValueMethods {
		value := "on"
		if !javaparser.HasJsonValue(m.Annotations) {
			value = "off"
		}
		got[m.Name] = value
	}
	want := map[string]string{"code": "off", "getWire": "on", "isActive": "off"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("methods = %v, want %v", got, want)
	}
	if len(enum.JsonValueFields) != 1 || enum.JsonValueFields[0].Name != "code" || !javaparser.HasJsonValue(enum.JsonValueFields[0].Annotations) {
		t.Errorf("fields = %+v, want the field code with @JsonValue", enum.JsonValueFields)
	}

	iface := newTypeScope(nil, nil).toJavaClass(ktparser.Class{
		IsInterface: true,
		Fields:      []ktparser.Property{testProperty("wire", "String", ktparser.Annotation{Name: "JsonValue", UseSiteTarget: "get"})},
		Methods:     []ktparser.Function{{Name: "code", Annotations: []ktparser.Annotation{off}}},
	})
	if len(iface.Methods) != 0 {
		t.Errorf("interface methods = %+v, want none", iface.Methods)
	}
	names := map[string]bool{}
	for _, m := range iface.JsonValueMethods {
		names[m.Name] = true
	}
	if len(iface.JsonValueMethods) != 2 || !names["getWire"] || !names["code"] || names["jsonValue"] {
		t.Errorf("interface members = %+v, want getWire and code", iface.JsonValueMethods)
	}
}

// The interfaces of an enum reach the shared resolver spelled the way the file of
// the enum binds them, as those of a class do: it reads the names through the
// imports of the enum, and a renaming import is not among them.
func TestToJavaClassesSpellsTheInterfacesOfAnEnumAsItsFileBindsThem(t *testing.T) {
	tests := []struct {
		name       string
		imports    []ktparser.Import
		interfaces []string
		declared   []ktparser.Class
		aliases    []ktparser.TypeAlias
		want       []string
	}{
		{name: "none"},
		{name: "a renaming import", imports: []ktparser.Import{{Path: "p.Valued", Alias: "V"}}, interfaces: []string{"V"}, want: []string{"p.Valued"}},
		{name: "a renaming import with a type argument", imports: []ktparser.Import{{Path: "p.Valued", Alias: "V"}}, interfaces: []string{"V<String>"}, want: []string{"p.Valued<java.lang.String>"}},
		{name: "a renaming import of a nested interface", imports: []ktparser.Import{{Path: "p.Outer.Valued", Alias: "V"}}, interfaces: []string{"V"}, want: []string{"p.Outer.Valued"}},
		{name: "a plain import", imports: []ktparser.Import{{Path: "p.Valued"}}, interfaces: []string{"Valued"}, want: []string{"Valued"}},
		{name: "an import on demand", imports: []ktparser.Import{{Path: "p", OnDemand: true}}, interfaces: []string{"Valued"}, want: []string{"Valued"}},
		{name: "a nested interface through its outer class", imports: []ktparser.Import{{Path: "p.Outer"}}, interfaces: []string{"Outer.Valued"}, want: []string{"Outer.Valued"}},
		{name: "a name written in full", interfaces: []string{"p.Valued"}, want: []string{"p.Valued"}},
		{name: "a name no file binds", interfaces: []string{"Valued"}, want: []string{"Valued"}},
		{
			name:       "an alias of the name of a declaration of the package",
			imports:    []ktparser.Import{{Path: "p.Valued", Alias: "Valued"}},
			interfaces: []string{"Valued"},
			declared:   []ktparser.Class{{Name: "Valued", Package: "app", IsInterface: true}},
			want:       []string{"p.Valued"},
		},
		{
			name:       "a typealias",
			interfaces: []string{"V"},
			aliases:    []ktparser.TypeAlias{{Name: "V", Package: "app", Target: "p.Valued"}},
			want:       []string{"p.Valued"},
		},
		{
			name:       "the order of the interfaces",
			imports:    []ktparser.Import{{Path: "p.Valued", Alias: "V"}, {Path: "q.Named", Alias: "N"}},
			interfaces: []string{"N", "java.io.Serializable", "V"},
			want:       []string{"q.Named", "java.io.Serializable", "p.Valued"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enum := ktparser.Class{
				Name: "State", Package: "app", FilePath: "State.kt", IsEnum: true, EnumValues: []string{"A"},
				Imports: tt.imports, Interfaces: tt.interfaces,
			}
			classes := append([]ktparser.Class{enum}, tt.declared...)
			var got []string
			for _, class := range ToJavaClasses(classes, nil, tt.aliases) {
				if class.Name == "State" {
					got = class.Interfaces
				}
			}
			if len(got) != len(tt.want) || (len(got) > 0 && !reflect.DeepEqual(got, tt.want)) {
				t.Errorf("interfaces = %q, want %q", got, tt.want)
			}
		})
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

// A member type a class inherits from a superclass or an interface shadows the
// enum of the package that carries its name, as one the class declares does, and an
// enum that a name written in a class of an unrelated package does not bind to is
// not taken for it.
func TestResolveFromEnumsInheritedAsMemberTypes(t *testing.T) {
	classes := []ktparser.Class{
		{Name: "Status", Package: "p", FilePath: "p/Status.kt", IsEnum: true, EnumValues: []string{"TOP"}},
		{Name: "Outer", Package: "p", FilePath: "p/Outer.kt"},
		{Name: "Status", Outer: "Outer", Package: "p", FilePath: "p/Outer.kt", IsEnum: true, EnumValues: []string{"OUTER"}},
		{Name: "HasStatus", Package: "p", FilePath: "p/HasStatus.kt", IsInterface: true},
		{Name: "Status", Outer: "HasStatus", Package: "p", FilePath: "p/HasStatus.kt", IsEnum: true, EnumValues: []string{"IFACE"}},
		{Name: "Sub", Package: "p", FilePath: "p/Sub.kt", SuperClass: "Outer", Fields: []ktparser.Property{testProperty("s", "Status")}},
		{Name: "ViaIface", Package: "p", FilePath: "p/ViaIface.kt", Interfaces: []string{"HasStatus"}, Fields: []ktparser.Property{testProperty("s", "Status")}},
		{Name: "Plain", Package: "p", FilePath: "p/Plain.kt", Fields: []ktparser.Property{testProperty("s", "Status")}},
		{Name: "Stranger", Package: "q", FilePath: "q/Stranger.kt", Fields: []ktparser.Property{testProperty("s", "Status")}},
	}
	r := New(classes)
	want := map[string][]string{"Sub": {"OUTER"}, "ViaIface": {"IFACE"}, "Plain": {"TOP"}, "Stranger": nil}
	for i := range classes {
		expected, ok := want[classes[i].Name]
		if !ok {
			continue
		}
		field := objectField(t, r.ResolveFrom(&classes[i], classes[i].Name, nil), "s")
		var got []string
		for _, option := range field.Options {
			got = append(got, option.Value.(string))
		}
		if !reflect.DeepEqual(got, expected) {
			t.Errorf("%s.s options = %v, want %v", classes[i].Name, got, expected)
		}
	}
}

// An enum is written as the member of an interface that carries @JsonValue, whether
// the member is a function or a property whose getter carries it, and not as one
// whose annotation lands elsewhere; an enum with an entry that @JsonProperty gives
// a name is not written as the names of its entries either. Both keep their type
// name.
func TestResolveFromEnumsWrittenByAnInterfaceAndRenamedEntries(t *testing.T) {
	function := ktparser.Function{Name: "code", Annotations: []ktparser.Annotation{testAnnotation("JsonValue", nil)}}
	property := testProperty("wire", "String", ktparser.Annotation{Name: "JsonValue", UseSiteTarget: "get"})
	elsewhere := testProperty("other", "String", ktparser.Annotation{Name: "JsonValue", UseSiteTarget: "param"})
	classes := []ktparser.Class{
		{Name: "ByFunction", Package: "p", IsInterface: true, Methods: []ktparser.Function{function}},
		{Name: "ByProperty", Package: "p", IsInterface: true, Fields: []ktparser.Property{property}},
		{Name: "ByNothing", Package: "p", IsInterface: true, Fields: []ktparser.Property{elsewhere}},
		{Name: "Deeper", Package: "p", IsInterface: true, Interfaces: []string{"ByFunction"}},
		{Name: "A", Package: "p", IsEnum: true, EnumValues: []string{"X"}, Interfaces: []string{"ByFunction"}},
		{Name: "B", Package: "p", IsEnum: true, EnumValues: []string{"X"}, Interfaces: []string{"ByProperty"}},
		{Name: "C", Package: "p", IsEnum: true, EnumValues: []string{"X"}, Interfaces: []string{"ByNothing"}},
		{Name: "D", Package: "p", IsEnum: true, EnumValues: []string{"X"}, Interfaces: []string{"Deeper"}},
		{Name: "E", Package: "p", IsEnum: true, EnumValues: []string{"X", "Y"}, EnumJsonProperty: true},
		{Name: "Holder", Package: "p", FilePath: "p/Holder.kt", Fields: []ktparser.Property{
			mutableProperty("a", "A"), mutableProperty("b", "B"), mutableProperty("c", "C"), mutableProperty("d", "D"), mutableProperty("e", "E"),
		}},
	}
	r := New(classes)
	holder := r.ResolveFrom(&classes[len(classes)-1], "Holder", nil)
	for name, want := range map[string]string{"a": "A", "b": "B", "d": "D", "e": "E"} {
		if got := objectField(t, holder, name); got.Model.TypeName != want || len(got.Options) != 0 {
			t.Errorf("%s = %+v, options %v, want the type name %s of an enum written in a form of its own", name, got.Model, got.Options, want)
		}
	}
	if c := objectField(t, holder, "c"); c.Model.TypeName != model.JsonTypeString || len(c.Options) != 1 {
		t.Errorf("c = %+v, options %v, want a string of X: @JsonValue on the parameter is not read as the value", c.Model, c.Options)
	}
	bound := r.ResolveForBinding(&classes[len(classes)-1], "Holder", nil)
	if got := objectField(t, bound, "e"); len(got.Options) != 2 || got.Options[1].Value != "Y" {
		t.Errorf("binding e options = %v, want X and Y", got.Options)
	}
}

// libraryEnums answers the dependency lookups of a Kotlin resolver with the classes
// it holds, by qualified name.
type libraryEnums map[string]javaparser.Class

func (l libraryEnums) ResolveClass(name string) *javaparser.Class {
	if c, ok := l[name]; ok {
		return &c
	}
	return nil
}

func entries(field *model.FieldModel) []string {
	var got []string
	for _, option := range field.Options {
		got = append(got, option.Value.(string))
	}
	return got
}

// Two enums of one simple name in different packages: each name takes the entries
// of the enum its scope gives it, in whatever order the declarations come in. An enum
// of the package of the source is the one the name denotes, and a class of another
// package that carries the name and that nothing imports is not taken for it, for
// the name written directly, in a collection or as a type argument.
func TestResolveFromEnumsOfOneNameInPackagesAreToldInAnyOrder(t *testing.T) {
	dto := ktparser.Class{Name: "Status", Package: "com.ex.app.dto", FilePath: "app/dto/Status.kt", IsEnum: true, EnumValues: []string{"OPEN", "CLOSED"}}
	lib := ktparser.Class{Name: "Status", Package: "com.ex.lib", FilePath: "lib/Status.kt", IsEnum: true, EnumValues: []string{"ON", "OFF"}}
	unrelated := ktparser.Class{Name: "Status", Package: "com.ex.unrelated", FilePath: "unrelated/Status.kt", Fields: []ktparser.Property{testProperty("unrelatedLabel", "String")}}
	box := ktparser.Class{Name: "Box", Package: "com.ex.gen", FilePath: "gen/Box.kt", TypeParameters: []string{"T"}, Fields: []ktparser.Property{testProperty("data", "T")}}
	holder := ktparser.Class{
		Name: "Holder", Package: "com.ex.app.dto", FilePath: "app/dto/Holder.kt",
		Imports: []ktparser.Import{{Path: "com.ex.gen.Box"}},
		Fields: []ktparser.Property{
			testProperty("own", "Status"), testProperty("ownLib", "com.ex.lib.Status"), testProperty("many", "List<Status>"),
			testProperty("boxed", "Box<Status>"), testProperty("boxedLib", "Box<com.ex.lib.Status>"),
		},
	}
	for name, classes := range map[string][]ktparser.Class{
		"as declared": {dto, lib, unrelated, box, holder},
		"reversed":    {holder, box, unrelated, lib, dto},
	} {
		r := New(classes)
		r.SetDependencyResolver(libraryEnums{})
		got := r.ResolveFrom(&holder, "Holder", nil)
		for field, want := range map[string][]string{"own": {"OPEN", "CLOSED"}, "ownLib": {"ON", "OFF"}, "many": {"OPEN", "CLOSED"}} {
			f := objectField(t, got, field)
			if !reflect.DeepEqual(entries(f), want) || (f.Model.Kind != model.KindArray && f.Model.TypeName != model.JsonTypeString) {
				t.Errorf("%s: Holder.%s = %+v, options %v, want a string of %v", name, field, f.Model, entries(f), want)
			}
		}
		for field, want := range map[string][]string{"boxed": {"OPEN", "CLOSED"}, "boxedLib": {"ON", "OFF"}} {
			f := objectField(t, objectField(t, got, field).Model, "data")
			if !reflect.DeepEqual(entries(f), want) {
				t.Errorf("%s: Holder.%s.data options = %v, want %v", name, field, entries(f), want)
			}
		}
	}
}

// An enum of the dependencies is not turned away because the project nests an enum
// of the same name in a class of the package the dependency declares it in: the
// enum nested in a class is not the enum of the package.
func TestResolveFromADependencyEnumIsNotTurnedAwayByANestedEnumOfItsName(t *testing.T) {
	classes := []ktparser.Class{
		{Name: "Outer", Package: "p", FilePath: "p/Outer.kt"},
		{Name: "Status", Outer: "Outer", Package: "p", FilePath: "p/Outer.kt", IsEnum: true, EnumValues: []string{"NESTED"}},
		{
			Name: "Holder", Package: "h", FilePath: "h/Holder.kt",
			Imports: []ktparser.Import{{Path: "p.Status"}},
			Fields:  []ktparser.Property{testProperty("status", "Status")},
		},
	}
	r := New(classes)
	r.SetDependencyResolver(libraryEnums{"p.Status": {Name: "Status", Package: "p", IsEnum: true, EnumConstants: []string{"LIB"}}})
	got := objectField(t, r.ResolveFrom(&classes[2], "Holder", nil), "status")
	if !reflect.DeepEqual(entries(got), []string{"LIB"}) {
		t.Errorf("Holder.status = %+v, options %v, want the enum of the dependencies, not the one nested in Outer", got.Model, entries(got))
	}
}

// A `@see` that no tier of the scope declares a name for takes the one enum of the
// project that carries the name, and none when two do; the scope comes first, so a
// class of the package that declares the name keeps it, and a field typed with the
// name is read strictly.
func TestResolveFromSeeNamesAnEnumOfAnotherPackageOnlyWhenItIsTheOne(t *testing.T) {
	doc := func(name, ref string) ktparser.Property {
		return ktparser.Property{Name: name, Type: "String", KDoc: "/** @see " + ref + " */"}
	}
	classes := []ktparser.Class{
		{Name: "Level", Package: "x", FilePath: "x/Level.kt", IsEnum: true, EnumValues: []string{"LOW", "HIGH"}},
		{Name: "Mode", Package: "y", FilePath: "y/Mode.kt", IsEnum: true, EnumValues: []string{"A"}},
		{Name: "Mode", Package: "z", FilePath: "z/Mode.kt", IsEnum: true, EnumValues: []string{"B"}},
		{Name: "Level", Package: "app", FilePath: "app/Level.kt"},
		{
			Name: "Doc", Package: "app", FilePath: "app/Doc.kt",
			Fields: []ktparser.Property{doc("shadowed", "Level"), doc("ambiguous", "Mode"), doc("qualified", "x.Level")},
		},
		{
			Name: "Doc2", Package: "b", FilePath: "b/Doc2.kt",
			Fields: []ktparser.Property{doc("level", "Level"), doc("ambiguous", "Mode"), testProperty("typed", "Level")},
		},
	}
	r := New(classes)
	for holder, want := range map[string]map[string][]string{
		"Doc":  {"shadowed": nil, "ambiguous": nil, "qualified": {"LOW", "HIGH"}},
		"Doc2": {"level": {"LOW", "HIGH"}, "ambiguous": nil, "typed": nil},
	} {
		for i := range classes {
			if classes[i].Name != holder {
				continue
			}
			got := r.ResolveFrom(&classes[i], holder, nil)
			for name, entries := range want {
				if field := objectField(t, got, name); !reflect.DeepEqual(entriesOf(field), entries) {
					t.Errorf("%s.%s options = %v, want %v", holder, name, entriesOf(field), entries)
				}
			}
		}
	}
}

func entriesOf(field *model.FieldModel) []string {
	var got []string
	for _, option := range field.Options {
		got = append(got, option.Value.(string))
	}
	return got
}

// An annotation class is not a type a name can denote. A bare name falls back
// to the class of another package, and a field typed with the name of an
// annotation in its own package does not take that annotation's members.
func TestResolveFromAnAnnotationClassIsNotATypeOfThatName(t *testing.T) {
	ann := ktparser.Class{
		Name: "Code", Package: "ann", FilePath: "ann/Code.kt", IsAnnotation: true,
		Fields: []ktparser.Property{testProperty("marker", "Int")},
	}
	codeClass := ktparser.Class{
		Name: "Code", Package: "lib", FilePath: "lib/Code.kt",
		Fields: []ktparser.Property{testProperty("value", "Int")},
	}
	holder := ktparser.Class{
		Name: "Holder", Package: "app", FilePath: "app/Holder.kt",
		Fields: []ktparser.Property{testProperty("code", "Code")},
	}
	sameAnn := ktparser.Class{
		Name: "Mark", Package: "app", FilePath: "app/Mark.kt", IsAnnotation: true,
		Fields: []ktparser.Property{testProperty("marker", "Int")},
	}
	same := ktparser.Class{
		Name: "Same", Package: "app", FilePath: "app/Same.kt",
		Fields: []ktparser.Property{testProperty("mark", "Mark")},
	}
	r := New([]ktparser.Class{ann, codeClass, holder, sameAnn, same})
	code := objectField(t, r.ResolveFrom(&holder, "Holder", nil), "code")
	if _, ok := code.Model.Fields["value"]; !ok || code.Model.Fields["marker"] != nil {
		t.Errorf("code = %+v, want the class lib.Code", code.Model)
	}
	mark := objectField(t, r.ResolveFrom(&same, "Same", nil), "mark")
	if mark.Model.Fields["marker"] != nil {
		t.Errorf("mark = %+v, want no member of the annotation class", mark.Model)
	}
}

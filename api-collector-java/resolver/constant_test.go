package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

func constant(name, initializer string) parser.Field {
	return parser.Field{Name: name, Type: "String", IsStatic: true, IsFinal: true, HasInitializer: true, Constant: initializer}
}

// constantLibrary answers dependency lookups from classes, and records them.
type constantLibrary struct {
	classes map[string]parser.Class
	queried []string
}

func (l *constantLibrary) ResolveClass(className string) *parser.Class {
	l.queried = append(l.queried, className)
	if c, ok := l.classes[className]; ok {
		return &c
	}
	return nil
}

func TestConstantValue(t *testing.T) {
	classes := []parser.Class{
		{
			Name: "Handler", Package: "com.example.app", FilePath: "Handler.java",
			SuperClass: "BaseHandler", Interfaces: []string{"Names"},
			Imports: []parser.Import{
				{Path: "com.example.shared.Headers"},
				{Path: "com.example.shared.Tenants.TENANT", Static: true},
				{Path: "com.example.shared.Regions", Static: true, OnDemand: true},
				{Path: "com.example.lib.LibHeaders"},
				// A Kotlin top-level constant: the import names a package
				// member, not a member of a class.
				{Path: "com.example.app.TOP"},
			},
			Fields: []parser.Field{constant("OWN", `"X-Own"`), {Name: "instance", Type: "String"}},
		},
		{Name: "BaseHandler", Package: "com.example.app", Fields: []parser.Field{constant("INHERITED", `"X-Inherited"`)}},
		{Name: "Names", Package: "com.example.app", IsInterface: true, Fields: []parser.Field{constant("FROM_INTERFACE", `"X-Interface"`)}},
		{Name: "Headers", Package: "com.example.shared", Fields: []parser.Field{
			constant("VERSION", `"X-Version"`),
			constant("ALIAS", "Base.VERSION"),
			constant("ESCAPED", `"a\"b"`),
			constant("KOTLIN_DOLLAR", `"\$x"`),
			constant("RAW", `"""raw"""`),
			constant("LOOP_A", "LOOP_B"),
			constant("LOOP_B", "LOOP_A"),
		}},
		{Name: "Base", Package: "com.example.shared", Fields: []parser.Field{constant("VERSION", `"X-Base-Version"`)}},
		{Name: "Tenants", Package: "com.example.shared", Fields: []parser.Field{constant("TENANT", `"X-Tenant"`)}},
		{Name: "Regions", Package: "com.example.shared", Fields: []parser.Field{constant("REGION", `"X-Region"`)}},
		{Name: "Outer", Package: "com.example.app"},
		// A nested class is registered under its package and simple name.
		{Name: "Inner", Package: "com.example.app", Fields: []parser.Field{constant("NESTED", `"X-Nested"`)}},
	}
	library := &constantLibrary{classes: map[string]parser.Class{
		"com.example.lib.LibHeaders": {Name: "LibHeaders", Package: "com.example.lib", Fields: []parser.Field{constant("CHANNEL", `"X-Channel"`)}},
	}}
	r := NewTypeResolver(classes)
	r.SetDependencyResolver(library)
	source := classes[0]

	tests := []struct {
		ref  string
		want string
		ok   bool
	}{
		{"OWN", "X-Own", true},
		{"INHERITED", "X-Inherited", true},
		{"FROM_INTERFACE", "X-Interface", true},
		{"TENANT", "X-Tenant", true},
		{"REGION", "X-Region", true},
		{"Headers.VERSION", "X-Version", true},
		{"Headers . VERSION", "X-Version", true},
		{"com.example.shared.Headers.VERSION", "X-Version", true},
		{"Headers.Companion.VERSION", "X-Version", true},
		{"Headers.ALIAS", "X-Base-Version", true},
		{"Headers.ESCAPED", `a"b`, true},
		{"Headers.KOTLIN_DOLLAR", "$x", true},
		{"Headers.RAW", "raw", true},
		{"Outer.Inner.NESTED", "X-Nested", true},
		{"LibHeaders.CHANNEL", "X-Channel", true},
		{"Headers.LOOP_A", "", false},
		{"Headers.MISSING", "", false},
		{"instance", "", false},
		{"Missing.NAME", "", false},
		{"TOP", "", false},
	}
	for _, tt := range tests {
		got, ok := r.ConstantValue(source, tt.ref)
		if got != tt.want || ok != tt.ok {
			t.Errorf("ConstantValue(%q) = %q, %v; want %q, %v", tt.ref, got, ok, tt.want, tt.ok)
		}
	}

	// A name that is not qualified, a package of the scanned sources and the
	// JDK are never asked of the dependencies.
	for _, q := range library.queried {
		switch q {
		case "com.example.lib.LibHeaders":
		default:
			t.Errorf("the dependencies were asked for %s", q)
		}
	}
}

func TestInheritedStaticFieldsAreNotProperties(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Base", Fields: []parser.Field{constant("TYPE", `"base"`), {Name: "id", Type: "Long"}}},
		{Name: "Child", SuperClass: "Base", Fields: []parser.Field{{Name: "name", Type: "String"}}},
	})
	got := r.Resolve("Child", nil)
	if got.Kind != model.KindObject {
		t.Fatalf("Child = %+v", got)
	}
	if _, ok := got.Fields["TYPE"]; ok {
		t.Errorf("fields = %v: a static field of the superclass is not instance state", got.OrderedFieldNames())
	}
	for _, name := range []string{"name", "id"} {
		if _, ok := got.Fields[name]; !ok {
			t.Errorf("fields = %v, want %s", got.OrderedFieldNames(), name)
		}
	}
}

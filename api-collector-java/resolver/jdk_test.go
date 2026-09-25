package resolver

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

func TestResolve_JdkValueTypesBecomeScalars(t *testing.T) {
	cases := []struct {
		rawType string
		want    string
	}{
		// java.time is a string on the wire, spelled with or without its package.
		{"LocalDateTime", model.JsonTypeString},
		{"java.time.LocalDateTime", model.JsonTypeString},
		{"LocalDate", model.JsonTypeString},
		{"Instant", model.JsonTypeString},
		{"ZonedDateTime", model.JsonTypeString},
		// java.math
		{"BigDecimal", model.JsonTypeDouble},
		{"java.math.BigDecimal", model.JsonTypeDouble},
		{"BigInteger", model.JsonTypeLong},
		// java.util
		{"Date", model.JsonTypeString},
		{"java.util.Date", model.JsonTypeString},
		{"UUID", model.JsonTypeString},
		// Ambiguous short names are only rewritten when fully qualified.
		{"java.sql.Time", model.JsonTypeString},
		{"java.sql.Timestamp", model.JsonTypeString},
		{"java.net.URL", model.JsonTypeString},
	}

	for _, tc := range cases {
		r := NewTypeResolver(nil)
		got := r.Resolve(tc.rawType, nil)
		if got.Kind != model.KindSingle || got.TypeName != tc.want {
			t.Errorf("Resolve(%q) = (%s, %q), want (single, %q)", tc.rawType, got.Kind, got.TypeName, tc.want)
		}
	}
}

func TestResolve_ObjectIsAnEmptyObject(t *testing.T) {
	r := NewTypeResolver(nil)

	for _, rawType := range []string{"Object", "java.lang.Object"} {
		got := r.Resolve(rawType, nil)
		if !got.IsObject() {
			t.Fatalf("Resolve(%q) kind = %s, want object", rawType, got.Kind)
		}
		if len(got.Fields) != 0 {
			t.Errorf("Resolve(%q) fields = %v, want empty", rawType, got.Fields)
		}
	}
}

func TestResolve_JdkTypesAreNotReportedUnresolved(t *testing.T) {
	sink := collector.NewUnresolvedSet()
	r := NewTypeResolver(nil)
	r.SetUnresolved(sink)

	r.Resolve("LocalDateTime", nil)
	r.Resolve("BigDecimal", nil)
	r.Resolve("Object", nil)

	if counts := sink.Counts(); len(counts) != 0 {
		t.Errorf("JDK value types are resolved, not failures: %v", counts)
	}
}

// A project class whose simple name collides with an ambiguous JDK short name
// must still resolve from the registry instead of being folded into a scalar.
func TestResolve_AmbiguousShortNamesStayResolvable(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{
			Name: "Time",
			Fields: []parser.Field{
				{Name: "value", Type: "String"},
			},
		},
	})

	got := r.Resolve("Time", nil)

	if !got.IsObject() {
		t.Fatalf("Resolve(Time) kind = %s, want object from the registry", got.Kind)
	}
}

func TestSimpleTypeNameStripsGenericArguments(t *testing.T) {
	cases := map[string]string{
		"List<Order>":              "list",
		"Map<String, List<Order>>": "map",
		"java.time.LocalDateTime":  "localdatetime",
		"PageResult":               "pageresult",
		"":                         "",
	}

	for raw, want := range cases {
		if got := simpleTypeName(raw); got != want {
			t.Errorf("simpleTypeName(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestResolve_ProjectClassWinsOverJdkLookup(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{
			Name: "Date",
			Fields: []parser.Field{
				{Name: "epochDay", Type: "long"},
			},
		},
	})

	got := r.Resolve("Date", nil)

	if !got.IsObject() {
		t.Fatalf("Resolve(Date) kind = %s, want object from the registry", got.Kind)
	}
}

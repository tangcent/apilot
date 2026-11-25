package resolver

import (
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// importing builds the compilation unit a type is written in: its package and
// the imports it names, a trailing `.*` standing for an import on demand.
func importing(pkg string, imports ...string) parser.Class {
	source := parser.Class{Name: "Holder", Package: pkg, FilePath: pkg + "/Holder.java"}
	for _, path := range imports {
		if len(path) > 2 && path[len(path)-2:] == ".*" {
			source.Imports = append(source.Imports, parser.Import{Path: path[:len(path)-2], OnDemand: true})
			continue
		}
		source.Imports = append(source.Imports, parser.Import{Path: path})
	}
	return source
}

func optionValues(options []model.FieldOption) []any {
	var values []any
	for _, option := range options {
		values = append(values, option.Value)
	}
	return values
}

// A collection or a map spelled with its package is the collection or the map
// its simple name denotes. Before, only the simple names were known, so a field
// declared as `java.util.List<String>` was one opaque value.
func TestResolve_QualifiedCollectionsAndMapsAreReadLikeTheSimpleOnes(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Order", Package: "com.example", Fields: []parser.Field{{Name: "id", Type: "long"}}},
	})
	r.SetUnresolved(collector.NewUnresolvedSet())

	for typeName, want := range map[string]string{
		"java.util.List<String>":                      "array<string>",
		"java.util.Set<Long>":                         "array<long>",
		"java.util.Collection<Integer>":               "array<int>",
		"java.lang.Iterable<String>":                  "array<string>",
		"java.util.ArrayList<String>":                 "array<string>",
		"java.util.LinkedList<String>":                "array<string>",
		"java.util.HashSet<String>":                   "array<string>",
		"java.util.Map<String, Long>":                 "map<long>",
		"java.util.HashMap<String, String>":           "map<string>",
		"java.util.LinkedHashMap<String, Integer>":    "map<int>",
		"java.util.TreeMap<String, Double>":           "map<double>",
		"java.util.List<com.example.Order>":           "array<object:Order>",
		"java.util.List<java.util.Map<String, Long>>": "array<map<long>>",
		"Map<String, java.util.Set<Long>>":            "map<array<long>>",
		"java.util.List<String>[]":                    "array<array<string>>",
		"Mono<java.util.List<String>>":                "array<string>",
		"ResponseEntity<java.util.Map<String, Long>>": "map<long>",
		"? extends java.util.List<String>":            "array<string>",
		// The simple names the other collectors of this repository already
		// read as collections and maps.
		"Iterable<String>":                  "array<string>",
		"LinkedHashSet<String>":             "array<string>",
		"java.util.LinkedHashSet<String>":   "array<string>",
		"SortedSet<String>":                 "array<string>",
		"TreeSet<String>":                   "array<string>",
		"SortedMap<String, Long>":           "map<long>",
		"java.util.SortedMap<String, Long>": "map<long>",
	} {
		if got := arrayShape(r.Resolve(typeName, nil)); got != want {
			t.Errorf("%s = %s, want %s", typeName, got, want)
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// Only the packages of the JDK qualify a collection: a class of another package
// that happens to be called List, and a nested class of a map, are not one.
func TestResolve_OnlyJavaUtilQualifiesACollection(t *testing.T) {
	r := NewTypeResolver(nil)
	r.SetUnresolved(collector.NewUnresolvedSet())

	for typeName, want := range map[string]string{
		"com.example.List<String>":          "com.example.List<String>",
		"java.util.Map.Entry<String, Long>": "java.util.Map.Entry<String, Long>",
	} {
		if got := arrayShape(r.Resolve(typeName, nil)); got != want {
			t.Errorf("%s = %s, want the opaque name", typeName, got)
		}
	}
}

// A class that extends a map spelled with its package is a map.
func TestResolve_ClassExtendingAQualifiedMapIsAMap(t *testing.T) {
	r := NewTypeResolver([]parser.Class{{
		Name:               "Registry",
		Package:            "com.example",
		SuperClass:         "java.util.HashMap<String, Long>",
		SuperClassTypeArgs: []string{"String", "Long"},
	}})

	if got := arrayShape(r.Resolve("com.example.Registry", nil)); got != "map<long>" {
		t.Errorf("Registry = %s, want map<long>", got)
	}
}

// An Optional is what it holds: Jackson's Jdk8Module, which Spring Boot
// registers, writes the value or null.
func TestResolve_OptionalIsTheValueItHolds(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Order", Package: "com.example", Fields: []parser.Field{{Name: "id", Type: "long"}}},
	})
	r.SetUnresolved(collector.NewUnresolvedSet())

	named := importing("com.example", "java.util.Optional", "reactor.core.publisher.Mono")
	onDemand := importing("com.example", "java.util.*")
	for _, tc := range []struct {
		source   parser.Class
		typeName string
		want     string
	}{
		{named, "Optional<String>", "string"},
		{named, "Optional<Long>", "long"},
		{named, "Optional<Order>", "object:Order"},
		{named, "Optional<List<String>>", "array<string>"},
		{named, "Optional<Map<String, Order>>", "map<object:Order>"},
		{named, "List<Optional<String>>", "array<string>"},
		{named, "Mono<Optional<Order>>", "object:Order"},
		{onDemand, "Optional<String>", "string"},
		// Written in full, the name needs no import.
		{importing("com.example"), "java.util.Optional<Order>", "object:Order"},
		{importing("com.example"), "java.util.Optional<java.util.List<String>>", "array<string>"},
		{named, "java.util.Optional<Long>", "long"},
	} {
		if got := arrayShape(r.ResolveFrom(tc.source, tc.typeName, nil)); got != tc.want {
			t.Errorf("%s = %s, want %s", tc.typeName, got, tc.want)
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// OptionalInt, OptionalLong and OptionalDouble hold a number or nothing.
func TestResolve_PrimitiveOptionalsAreNumbers(t *testing.T) {
	r := NewTypeResolver(nil)
	r.SetUnresolved(collector.NewUnresolvedSet())

	source := importing("com.example", "java.util.OptionalInt", "java.util.OptionalLong", "java.util.*")
	for typeName, want := range map[string]string{
		"OptionalInt":              "int",
		"OptionalLong":             "long",
		"OptionalDouble":           "double",
		"java.util.OptionalDouble": "double",
		"List<OptionalInt>":        "array<int>",
	} {
		if got := arrayShape(r.ResolveFrom(source, typeName, nil)); got != want {
			t.Errorf("%s = %s, want %s", typeName, got, want)
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// What an Optional stands for is decided by the scope of its name: a class of
// the project or of another library is a class, and a bare name nothing
// imports is not known at all.
func TestResolve_OnlyJavaUtilOptionalIsTransparent(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Optional", Package: "com.example.own", TypeParameters: []string{"T"},
			Fields: []parser.Field{{Name: "present", Type: "boolean"}}},
	})
	r.SetUnresolved(collector.NewUnresolvedSet())

	// A project class of that name, in the package of the source, wins.
	if got := arrayShape(r.ResolveFrom(importing("com.example.own", "java.util.*"), "Optional<String>", nil)); got != "object:Optional" {
		t.Errorf("project Optional = %s, want object:Optional", got)
	}
	// Guava's is imported by name, so it is not java.util's.
	if got := arrayShape(r.ResolveFrom(importing("com.example", "com.google.common.base.Optional"), "Optional<String>", nil)); got != "Optional<String>" {
		t.Errorf("Guava Optional = %s, want the opaque name", got)
	}
	// Nothing imports the name, and nothing in the project declares it.
	r = NewTypeResolver(nil)
	r.SetUnresolved(collector.NewUnresolvedSet())
	if got := arrayShape(r.ResolveFrom(importing("com.example"), "Optional<String>", nil)); got != "Optional<String>" {
		t.Errorf("unimported Optional = %s, want the opaque name", got)
	}
}

// A property that holds a value or nothing may be left out, whatever its
// annotations or the mode say about the value.
func TestResolve_OptionalPropertyIsNotRequired(t *testing.T) {
	notNull := []parser.Annotation{{Name: "NotNull"}}
	form := parser.Class{
		Name:    "Form",
		Package: "com.example",
		Imports: []parser.Import{{Path: "java.util.Optional"}, {Path: "java.util.OptionalInt"}},
		Fields: []parser.Field{
			{Name: "plain", Type: "String"},
			{Name: "constrained", Type: "String", Annotations: notNull},
			{Name: "optional", Type: "Optional<String>"},
			{Name: "constrainedOptional", Type: "Optional<String>", Annotations: notNull},
			{Name: "optionalInt", Type: "OptionalInt"},
		},
	}
	for mode, want := range map[string]map[string]bool{
		collector.RequiredFieldsDefault: {
			"plain": true, "constrained": true, "optional": false, "constrainedOptional": false, "optionalInt": false,
		},
		collector.RequiredFieldsConstraints: {
			"plain": false, "constrained": true, "optional": false, "constrainedOptional": false, "optionalInt": false,
		},
	} {
		r := NewTypeResolver([]parser.Class{form})
		r.SetRequiredFields(mode)
		got := map[string]bool{}
		for name, field := range r.Resolve("com.example.Form", nil).Fields {
			got[name] = field.Required
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s mode: required = %v, want %v", mode, got, want)
		}
	}
}

// Text, locale and currency values are strings on the wire, and so are the
// counters of java.util.concurrent.atomic. Each name is also asserted to be
// resolved: where a name has no entry the resolver reports it.
func TestResolve_JdkValueTypesAreScalars(t *testing.T) {
	r := NewTypeResolver(nil)
	r.SetUnresolved(collector.NewUnresolvedSet())

	source := importing("com.example", "java.util.Locale", "java.util.*", "java.sql.*", "java.time.ZoneId", "java.util.concurrent.atomic.*")
	for typeName, want := range map[string]string{
		"CharSequence":                          "string",
		"java.lang.CharSequence":                "string",
		"kotlin.CharSequence":                   "string",
		"Locale":                                "string",
		"java.util.Locale":                      "string",
		"Currency":                              "string",
		"java.util.Currency":                    "string",
		"TimeZone":                              "string",
		"java.util.TimeZone":                    "string",
		"ZoneId":                                "string",
		"java.time.ZoneId":                      "string",
		"AtomicInteger":                         "int",
		"AtomicLong":                            "long",
		"AtomicBoolean":                         "boolean",
		"LongAdder":                             "long",
		"java.util.concurrent.atomic.LongAdder": "long",
		"List<Locale>":                          "array<string>",
		"Map<String, Currency>":                 "map<string>",
		"java.sql.Timestamp":                    "string",
		"java.sql.Date":                         "string",
		"java.sql.Time":                         "string",
	} {
		if got := arrayShape(r.ResolveFrom(source, typeName, nil)); got != want {
			t.Errorf("%s = %s, want %s", typeName, got, want)
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// Locale, Currency and TimeZone are names a project declares as well, as
// Timestamp is: a bare name is the JDK class only when the source imports it,
// by name or with its package on demand. A class of the project wins.
func TestResolve_BareLocaleNamesNeedAnImport(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Currency", Package: "com.example.money", FilePath: "com/example/money/Currency.java",
			Fields: []parser.Field{{Name: "code", Type: "String"}}},
	})
	r.SetUnresolved(collector.NewUnresolvedSet())

	for _, tc := range []struct {
		name     string
		source   parser.Class
		typeName string
		want     string
	}{
		{"Locale imported by name", importing("com.example", "java.util.Locale"), "Locale", "string"},
		{"Locale imported on demand", importing("com.example", "java.util.*"), "Locale", "string"},
		{"TimeZone imported on demand", importing("com.example", "java.util.*"), "TimeZone", "string"},
		{"Locale not imported", importing("com.example"), "Locale", "Locale"},
		{"TimeZone of another package", importing("com.example", "org.joda.time.TimeZone"), "TimeZone", "TimeZone"},
		{"Currency declared in the package of the source", importing("com.example.money", "java.util.*"), "Currency", "object:Currency"},
		{"Currency imported by name from the project", importing("com.example", "com.example.money.Currency"), "Currency", "object:Currency"},
	} {
		if got := arrayShape(r.ResolveFrom(tc.source, tc.typeName, nil)); got != tc.want {
			t.Errorf("%s: %s = %s, want %s", tc.name, tc.typeName, got, tc.want)
		}
	}
}

// A java.sql type is a string, as a java.util.Date is. `Timestamp` and `Time`
// are names a project is as likely to declare, so the bare name is the JDK
// class only when the source imports it by name or imports java.sql on demand.
func TestResolve_BareSqlNamesNeedAnImport(t *testing.T) {
	r := NewTypeResolver(nil)
	r.SetUnresolved(collector.NewUnresolvedSet())

	for _, tc := range []struct {
		name   string
		source parser.Class
		want   string
	}{
		{"imported by name", importing("com.example", "java.sql.Timestamp"), "string"},
		{"imported on demand", importing("com.example", "java.sql.*"), "string"},
		{"imported on demand with others", importing("com.example", "java.util.List", "java.sql.*"), "string"},
		{"not imported", importing("com.example"), "Timestamp"},
		{"another package on demand", importing("com.example", "com.example.clock.*"), "Timestamp"},
		{"another Timestamp imported by name", importing("com.example", "com.example.clock.Timestamp"), "Timestamp"},
	} {
		if got := arrayShape(r.ResolveFrom(tc.source, "Timestamp", nil)); got != tc.want {
			t.Errorf("%s: Timestamp = %s, want %s", tc.name, got, tc.want)
		}
	}
	for typeName, want := range map[string]string{"Time": "string", "Date": "string", "Timestamp": "string"} {
		if got := arrayShape(r.ResolveFrom(importing("com.example", "java.sql.*"), typeName, nil)); got != want {
			t.Errorf("java.sql.* %s = %s, want %s", typeName, got, want)
		}
	}
	if got := arrayShape(r.ResolveFrom(importing("com.example"), "Time", nil)); got != "Time" {
		t.Errorf("unimported Time = %s, want the opaque name", got)
	}
}

// The rule is that of the whole qualified table, not of java.sql alone.
func TestResolve_BareNamesOfTheQualifiedTableNeedAnImport(t *testing.T) {
	r := NewTypeResolver(nil)
	r.SetUnresolved(collector.NewUnresolvedSet())

	if got := arrayShape(r.ResolveFrom(importing("com.example", "java.net.*"), "URL", nil)); got != "string" {
		t.Errorf("URL with java.net.* = %s, want string", got)
	}
	if got := arrayShape(r.ResolveFrom(importing("com.example", "java.net.URI"), "URI", nil)); got != "string" {
		t.Errorf("URI imported by name = %s, want string", got)
	}
	if got := arrayShape(r.ResolveFrom(importing("com.example"), "URL", nil)); got != "URL" {
		t.Errorf("unimported URL = %s, want the opaque name", got)
	}
}

// A project class of the same simple name wins over the java.sql type, however
// the source imports.
func TestResolve_ProjectClassWinsOverSqlNames(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Timestamp", Package: "com.example", FilePath: "com/example/Timestamp.java",
			Fields: []parser.Field{{Name: "epoch", Type: "long"}}},
		{Name: "Time", Package: "com.example.clock", FilePath: "com/example/clock/Time.java",
			Fields: []parser.Field{{Name: "hour", Type: "int"}}},
	})

	// Declared in the package of the source, which shadows an import on demand.
	if got := arrayShape(r.ResolveFrom(importing("com.example", "java.sql.*"), "Timestamp", nil)); got != "object:Timestamp" {
		t.Errorf("same-package Timestamp = %s, want object:Timestamp", got)
	}
	// Imported by name, which the qualified table answers for only when it is the JDK's.
	if got := arrayShape(r.ResolveFrom(importing("com.example.other", "com.example.clock.Time", "java.sql.*"), "Time", nil)); got != "object:Time" {
		t.Errorf("imported Time = %s, want object:Time", got)
	}
}

// A JDK enum is a string holding one of its constants, in the view that reads
// request parameters and in the one that writes the body.
func TestResolve_JdkEnumsAreStringsWithTheirConstants(t *testing.T) {
	days := []any{"MONDAY", "TUESDAY", "WEDNESDAY", "THURSDAY", "FRIDAY", "SATURDAY", "SUNDAY"}
	months := []any{
		"JANUARY", "FEBRUARY", "MARCH", "APRIL", "MAY", "JUNE",
		"JULY", "AUGUST", "SEPTEMBER", "OCTOBER", "NOVEMBER", "DECEMBER",
	}
	form := parser.Class{
		Name:    "Form",
		Package: "com.example",
		Imports: []parser.Import{{Path: "java.time.DayOfWeek"}, {Path: "java.time.Month"}, {Path: "java.util.Optional"}},
		Fields: []parser.Field{
			{Name: "day", Type: "DayOfWeek"},
			{Name: "month", Type: "java.time.Month"},
			{Name: "maybeMonth", Type: "Optional<Month>"},
			{Name: "days", Type: "List<DayOfWeek>"},
		},
	}
	r := NewTypeResolver([]parser.Class{form})
	r.SetUnresolved(collector.NewUnresolvedSet())

	wants := []struct {
		field   string
		shape   string
		options []any
	}{
		{"day", "string", days},
		{"month", "string", months},
		{"maybeMonth", "string", months},
		// The constants of a collection's elements are not attached, as they
		// are not for a project's enum.
		{"days", "array<string>", nil},
	}
	for view, resolve := range map[string]func() *model.ObjectModel{
		"json":    func() *model.ObjectModel { return r.Resolve("com.example.Form", nil) },
		"binding": func() *model.ObjectModel { return r.ResolveForBinding(parser.Class{}, "com.example.Form", nil) },
	} {
		fields := resolve().Fields
		for _, want := range wants {
			field := fields[want.field]
			if got := arrayShape(field.Model); got != want.shape {
				t.Errorf("%s view: %s = %s, want %s", view, want.field, got, want.shape)
			}
			if got := optionValues(field.Options); !reflect.DeepEqual(got, want.options) {
				t.Errorf("%s view: %s options = %v, want %v", view, want.field, got, want.options)
			}
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// Month and DayOfWeek are names a project declares as well. The bare name is
// the JDK enum only when the source imports it; a project's enum keeps its own
// constants, and one the JSON view does not read, for it declares @JsonValue, is
// left as it was, not given the JDK's.
func TestResolve_BareMonthNeedsAnImportAndYieldsToAProjectEnum(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "Month", Package: "com.example.fiscal", FilePath: "com/example/fiscal/Month.java",
			IsEnum: true, EnumConstants: []string{"Q1", "Q2"}, EnumJsonValue: true},
	})
	r.SetUnresolved(collector.NewUnresolvedSet())

	// The JDK's Month, imported by name.
	if got := arrayShape(r.ResolveFrom(importing("com.example", "java.time.Month"), "Month", nil)); got != "string" {
		t.Errorf("java.time.Month = %s, want string", got)
	}
	// The project's enum, imported by name: the JDK table does not answer, and
	// the JSON view gives an enum the opaque name it gives a type it cannot see.
	got := arrayShape(r.ResolveFrom(importing("com.example", "com.example.fiscal.Month"), "Month", nil))
	if got != "Month" {
		t.Errorf("project Month = %s, want the opaque name", got)
	}
	// The enum shares the package of the source and java.time is imported on
	// demand: the declaration shadows the import, though the JSON view cannot see it.
	got = arrayShape(r.ResolveFrom(importing("com.example.fiscal", "java.time.*"), "Month", nil))
	if got != "Month" {
		t.Errorf("shadowed Month = %s, want the opaque name", got)
	}
	// Nothing imports the name.
	if got := arrayShape(r.ResolveFrom(importing("com.example.other"), "DayOfWeek", nil)); got != "DayOfWeek" {
		t.Errorf("unimported DayOfWeek = %s, want the opaque name", got)
	}

	// The binding view reads the project's enum and gives it its own constants.
	fiscal := parser.Class{
		Name: "Report", Package: "com.example.fiscal", FilePath: "com/example/fiscal/Report.java",
		Imports: []parser.Import{{Path: "java.time", OnDemand: true}},
		Fields:  []parser.Field{{Name: "month", Type: "Month"}},
	}
	r = NewTypeResolver([]parser.Class{
		fiscal,
		{Name: "Month", Package: "com.example.fiscal", FilePath: "com/example/fiscal/Month.java",
			IsEnum: true, EnumConstants: []string{"Q1", "Q2"}},
	})
	month := r.ResolveForBinding(parser.Class{}, "com.example.fiscal.Report", nil).Fields["month"]
	if got := optionValues(month.Options); !reflect.DeepEqual(got, []any{"Q1", "Q2"}) {
		t.Errorf("binding view of the project Month: options = %v, want Q1 and Q2", got)
	}
}

// A Jackson tree is free-form, and a multipart file keeps its name as its type:
// that is how a handler parameter or a bound property is told to be a file.
func TestResolve_LibraryTypes(t *testing.T) {
	r := NewTypeResolver(nil)
	r.SetUnresolved(collector.NewUnresolvedSet())

	named := importing("com.example",
		"com.fasterxml.jackson.databind.JsonNode",
		"com.fasterxml.jackson.databind.node.ObjectNode",
		"com.fasterxml.jackson.databind.node.ArrayNode",
		"org.springframework.web.multipart.MultipartFile")
	onDemand := importing("com.example",
		"com.fasterxml.jackson.databind.*",
		"com.fasterxml.jackson.databind.node.*",
		"org.springframework.web.multipart.*")
	for _, source := range []parser.Class{named, onDemand, importing("com.example")} {
		for typeName, want := range map[string]string{
			"com.fasterxml.jackson.databind.JsonNode":         "object:object",
			"com.fasterxml.jackson.databind.node.ObjectNode":  "object:object",
			"com.fasterxml.jackson.databind.node.ArrayNode":   "array<object:object>",
			"org.springframework.web.multipart.MultipartFile": "MultipartFile",
		} {
			if got := arrayShape(r.ResolveFrom(source, typeName, nil)); got != want {
				t.Errorf("%s = %s, want %s", typeName, got, want)
			}
		}
	}
	for _, source := range []parser.Class{named, onDemand} {
		for typeName, want := range map[string]string{
			"JsonNode":              "object:object",
			"ObjectNode":            "object:object",
			"ArrayNode":             "array<object:object>",
			"MultipartFile":         "MultipartFile",
			"List<MultipartFile>":   "array<MultipartFile>",
			"MultipartFile[]":       "array<MultipartFile>",
			"Map<String, JsonNode>": "map<object:object>",
		} {
			if got := arrayShape(r.ResolveFrom(source, typeName, nil)); got != want {
				t.Errorf("%s = %s, want %s", typeName, got, want)
			}
		}
	}
	if unresolved := r.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}

	// The file is the model the resolver remembers, whatever the name.
	for _, source := range []parser.Class{named, onDemand} {
		if file := r.ResolveFrom(source, "MultipartFile", nil); !r.IsUpload(file) {
			t.Errorf("MultipartFile = %+v, want the model of an upload", file)
		}
		if files := r.ResolveFrom(source, "List<MultipartFile>", nil); !r.IsUpload(files.Items) {
			t.Errorf("List<MultipartFile> = %+v, want an array of uploads", files)
		}
	}

	// Without an import the bare name is not known to be Jackson's or Spring's.
	for _, typeName := range []string{"JsonNode", "MultipartFile"} {
		unimported := r.ResolveFrom(importing("com.example"), typeName, nil)
		if got := arrayShape(unimported); got != typeName {
			t.Errorf("unimported %s = %s, want the opaque name", typeName, got)
		}
		if r.IsUpload(unimported) {
			t.Errorf("unimported %s is an upload: nothing says it is Spring's", typeName)
		}
	}
}

// A project class named like a library class is the class: a tree of its own
// called JsonNode keeps its fields.
func TestResolve_ProjectClassWinsOverLibraryTypes(t *testing.T) {
	r := NewTypeResolver([]parser.Class{
		{Name: "JsonNode", Package: "com.example.tree", FilePath: "com/example/tree/JsonNode.java",
			Fields: []parser.Field{{Name: "label", Type: "String"}}},
	})

	for _, source := range []parser.Class{
		importing("com.example.tree", "com.fasterxml.jackson.databind.*"),
		importing("com.example", "com.example.tree.JsonNode"),
	} {
		if got := arrayShape(r.ResolveFrom(source, "JsonNode", nil)); got != "object:JsonNode" {
			t.Errorf("JsonNode from %v = %s, want the project class", source.Imports, got)
		}
	}
}

// The answers for a JDK enum and a library class do not depend on what the
// project's dependencies hold: a dependency's copy of a Jackson tree is its
// internals, not what is written.
func TestResolve_PlatformTypesAreAnsweredBeforeDependencies(t *testing.T) {
	r := NewTypeResolver(nil)
	r.SetDependencyResolver(dependencyStub{
		"com.fasterxml.jackson.databind.JsonNode": {
			Name: "JsonNode", Package: "com.fasterxml.jackson.databind",
			Fields: []parser.Field{{Name: "internals", Type: "String"}},
		},
		"java.time.Month": {
			Name: "Month", Package: "java.time", IsEnum: true, EnumConstants: []string{"X"},
		},
	})

	source := importing("com.example", "com.fasterxml.jackson.databind.JsonNode", "java.time.Month")
	if got := arrayShape(r.ResolveFrom(source, "JsonNode", nil)); got != "object:object" {
		t.Errorf("JsonNode = %s, want the free-form object", got)
	}
	if got := arrayShape(r.ResolveFrom(source, "Month", nil)); got != "string" {
		t.Errorf("Month = %s, want string", got)
	}
}

type dependencyStub map[string]parser.Class

func (d dependencyStub) ResolveClass(className string) *parser.Class {
	if class, ok := d[className]; ok {
		return &class
	}
	return nil
}

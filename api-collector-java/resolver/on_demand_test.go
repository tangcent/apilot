package resolver

import (
	"reflect"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// onDemand builds the import-on-demand declarations of a test source.
func onDemand(packages ...string) []parser.Import {
	var imports []parser.Import
	for _, pkg := range packages {
		imports = append(imports, parser.Import{Path: pkg, OnDemand: true})
	}
	return imports
}

func TestResolve_AnImportOnDemandNamesAClassOfTheDependencies(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"com.example.lib.model.OrderDraft": {
			Name: "OrderDraft", Package: "com.example.lib.model",
			Fields: []parser.Field{
				{Name: "no", Type: "String"},
				// Declared in the package of OrderDraft: no import names them.
				{Name: "address", Type: "Address"},
				{Name: "items", Type: "List<LineItem>"},
			},
		},
		"com.example.lib.model.Address":  {Name: "Address", Package: "com.example.lib.model", Fields: []parser.Field{{Name: "city", Type: "String"}}},
		"com.example.lib.model.LineItem": {Name: "LineItem", Package: "com.example.lib.model", Fields: []parser.Field{{Name: "sku", Type: "String"}}},
	}}
	controller := parser.Class{
		Name: "OrderController", Package: "com.example.app", FilePath: "OrderController.java",
		Imports: onDemand("java.util", "com.example.lib.model"),
	}
	report := parser.Class{
		Name: "ReportController", Package: "com.example.app", FilePath: "ReportController.java",
		Imports: onDemand("com.example.lib.model"),
	}
	unresolved := collector.NewUnresolvedSet()
	r := NewTypeResolver([]parser.Class{controller, report})
	r.SetDependencyResolver(library)
	r.SetUnresolved(unresolved)

	got := r.ResolveFrom(controller, "OrderDraft", nil)
	assertFieldNames(t, fieldNamesOf(t, r, controller, "OrderDraft"), []string{"address", "items", "no"})
	if address := got.Fields["address"].Model; address.Kind != model.KindObject || address.Fields["city"] == nil {
		t.Errorf("address = %+v, want Address expanded", address)
	}
	if items := got.Fields["items"].Model; items.Kind != model.KindArray || items.Items.Fields["sku"] == nil {
		t.Errorf("items = %+v, want an array of LineItem", items)
	}

	// The package of the controller is a package of the scanned sources, and a
	// platform package is a platform's: neither is asked. The package of the
	// library class is, for the classes next to it.
	want := []string{
		"com.example.lib.model.OrderDraft",
		"com.example.lib.model.Address", "com.example.lib.model.LineItem",
	}
	if !reflect.DeepEqual(library.asked, want) {
		t.Errorf("asked = %v, want %v", library.asked, want)
	}

	// Another source that names the class the same way, or the same way in the
	// same package, is answered without asking again.
	assertFieldNames(t, fieldNamesOf(t, r, report, "OrderDraft"), []string{"address", "items", "no"})
	if !reflect.DeepEqual(library.asked, want) {
		t.Errorf("asked after a second source = %v, want %v", library.asked, want)
	}
	if counts := unresolved.Counts(); len(counts) != 0 {
		t.Errorf("unresolved = %v", counts)
	}
}

// The package of a source that no declaration of the scanned sources shares is
// the package of a class read from a library: its neighbours need no import.
func TestResolve_APackageOfTheSourceComesBeforeAnImportOnDemand(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"com.example.lib.Shared":     {Name: "Shared", Package: "com.example.lib", Fields: []parser.Field{{Name: "imported", Type: "String"}}},
		"com.example.sibling.Shared": {Name: "Shared", Package: "com.example.sibling", Fields: []parser.Field{{Name: "samePackage", Type: "String"}}},
	}}
	// Read from a library: not a source of the project.
	source := parser.Class{Name: "Holder", Package: "com.example.sibling", Imports: onDemand("com.example.lib")}
	r := NewTypeResolver(nil)
	r.SetDependencyResolver(library)

	assertFieldNames(t, fieldNamesOf(t, r, source, "Shared"), []string{"samePackage"})
	if want := []string{"com.example.sibling.Shared"}; !reflect.DeepEqual(library.asked, want) {
		t.Errorf("asked = %v, want %v: nothing is asked once the package answers", library.asked, want)
	}
}

// A package that the scanned sources declare is the project's own: a source of
// the project does not ask the dependencies for a name its classes do not carry,
// whether the package is the source's own or is imported on demand. The price is
// a split package, a library class in a package of the project, which a project
// source does not find. A class read from a library asks for every package, its
// own included: the project's declaring the package too does not hide its
// neighbours.
func TestResolve_APackageOfTheScannedSourcesIsNotAsked(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"com.example.app.Shared":  {Name: "Shared", Package: "com.example.app", Fields: []parser.Field{{Name: "split", Type: "String"}}},
		"com.example.model.Thing": {Name: "Thing", Package: "com.example.model", Fields: []parser.Field{{Name: "split", Type: "String"}}},
		"com.example.lib.Thing":   {Name: "Thing", Package: "com.example.lib", Fields: []parser.Field{{Name: "imported", Type: "String"}}},
	}}
	source := parser.Class{Name: "Holder", Package: "com.example.app", FilePath: "Holder.java", Imports: onDemand("com.example.model", "com.example.lib")}
	r := NewTypeResolver([]parser.Class{
		source,
		{Name: "Order", Package: "com.example.model", FilePath: "Order.java"},
	})
	r.SetDependencyResolver(library)

	// Only the package nobody scanned is asked, and answers.
	assertFieldNames(t, fieldNamesOf(t, r, source, "Thing"), []string{"imported"})
	if want := []string{"com.example.lib.Thing"}; !reflect.DeepEqual(library.asked, want) {
		t.Errorf("asked = %v, want %v", library.asked, want)
	}
	// The library's class in a package of the project is not found.
	if shared := r.ResolveFrom(source, "Shared", nil); shared.Kind != model.KindSingle {
		t.Errorf("Shared = %+v, want an opaque value", shared)
	}
	for _, name := range library.asked {
		if name == "com.example.app.Shared" || name == "com.example.model.Thing" {
			t.Errorf("%s, a package of the scanned sources, was asked for: %v", name, library.asked)
		}
	}

	// A class read from a library, in the same package, finds it.
	neighbour := parser.Class{Name: "Neighbour", Package: "com.example.app", Imports: onDemand("com.example.lib")}
	assertFieldNames(t, fieldNamesOf(t, r, neighbour, "Shared"), []string{"split"})

	// The library's class is read now, and the project's source still does not
	// see it by its simple name: the answer does not depend on what was read.
	if shared := r.ResolveFrom(source, "Shared", nil); shared.Kind != model.KindSingle {
		t.Errorf("Shared after it was read = %+v, want an opaque value", shared)
	}
}

func TestResolve_ImportsOnDemandAreTriedInTheOrderTheyAreWritten(t *testing.T) {
	classes := map[string]parser.Class{
		"com.example.a.Thing": {Name: "Thing", Package: "com.example.a", Fields: []parser.Field{{Name: "fromA", Type: "String"}}},
		"com.example.b.Thing": {Name: "Thing", Package: "com.example.b", Fields: []parser.Field{{Name: "fromB", Type: "String"}}},
		"com.example.b.Other": {Name: "Other", Package: "com.example.b", Fields: []parser.Field{{Name: "other", Type: "String"}}},
	}
	for _, tc := range []struct {
		imports []string
		want    string
		asked   []string
	}{
		{[]string{"com.example.a", "com.example.b"}, "fromA", []string{"com.example.app.Thing", "com.example.a.Thing"}},
		{[]string{"com.example.b", "com.example.a"}, "fromB", []string{"com.example.app.Thing", "com.example.b.Thing"}},
	} {
		library := &libraryClasses{classes: classes}
		// Read from a library, so its own package is asked as well.
		source := parser.Class{Name: "Holder", Package: "com.example.app", Imports: onDemand(tc.imports...)}
		r := NewTypeResolver(nil)
		r.SetDependencyResolver(library)

		assertFieldNames(t, fieldNamesOf(t, r, source, "Thing"), []string{tc.want})
		if !reflect.DeepEqual(library.asked, tc.asked) {
			t.Errorf("imports %v: asked = %v, want %v: the first package that answers ends the search", tc.imports, library.asked, tc.asked)
		}
		// A name only the later package declares is found there.
		assertFieldNames(t, fieldNamesOf(t, r, source, "Other"), []string{"other"})
	}
}

func TestResolve_NoLookupForAPlatformPackageAStaticImportOrAKnownName(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{}}
	// Read from a library, so its own package may be asked.
	source := parser.Class{
		Name: "Holder", Package: "com.example.app", TypeParameters: []string{"Item"},
		Imports: append(onDemand("java.util", "javax.money", "kotlin.collections", "com.example.lib.Outer"),
			parser.Import{Path: "com.example.lib.Statics", OnDemand: true, Static: true}),
	}
	r := NewTypeResolver(nil)
	r.SetDependencyResolver(library)

	// A type variable of the source and the names the resolver answers for before
	// it consults a scope are not asked for.
	for _, name := range []string{"Item", "String", "Integer", "List", "Map"} {
		r.ResolveFrom(source, name, nil)
	}
	if len(library.asked) != 0 {
		t.Errorf("asked = %v, want nothing", library.asked)
	}
	// The names of JDK value types are answered for after the scope, so a class
	// of the package of the source or of one that an import on demand brings in is
	// what they denote: asked there, though not in a platform package, and neither
	// as written. The platform packages and the static import give no candidate;
	// javax.money is a library like any other. The class import names the types
	// nested in Outer, so its candidate is a nested one, and Outer itself is asked
	// for once, for the member types it inherits.
	for _, name := range []string{"Object", "Date", "UUID"} {
		r.ResolveFrom(source, name, nil)
	}
	want := []string{
		"com.example.app.Object", "javax.money.Object", "com.example.lib.Outer.Object", "com.example.lib.Outer",
		"com.example.app.Date", "javax.money.Date", "com.example.lib.Outer.Date",
		"com.example.app.UUID", "javax.money.UUID", "com.example.lib.Outer.UUID",
	}
	if !reflect.DeepEqual(library.asked, want) {
		t.Errorf("asked = %v, want %v", library.asked, want)
	}
	// Any other name is asked as written too, for a resolver that knows classes
	// by their simple names.
	library.asked = nil
	r.ResolveFrom(source, "Unknown", nil)
	if want := []string{"com.example.app.Unknown", "javax.money.Unknown", "com.example.lib.Outer.Unknown", "Unknown"}; !reflect.DeepEqual(library.asked, want) {
		t.Errorf("asked = %v, want %v", library.asked, want)
	}
}

func TestResolve_ACandidateIsAskedOnceHoweverOftenTheNameIsMet(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{}}
	first := parser.Class{Name: "First", Package: "com.example.app", FilePath: "First.java", Imports: onDemand("com.example.lib")}
	second := parser.Class{Name: "Second", Package: "com.example.web", FilePath: "Second.java", Imports: onDemand("com.example.lib", "com.example.other")}
	r := NewTypeResolver([]parser.Class{first, second})
	r.SetDependencyResolver(library)

	for i := 0; i < 3; i++ {
		r.ResolveFrom(first, "Missing", nil)
		r.ResolveFrom(second, "Missing", nil)
		r.ResolveFrom(second, "List<Missing>", nil)
	}
	// The packages of the two sources are packages of the scanned sources.
	want := []string{"com.example.lib.Missing", "Missing", "com.example.other.Missing"}
	if !reflect.DeepEqual(library.asked, want) {
		t.Errorf("asked = %v, want %v", library.asked, want)
	}
}

// A type argument is bound to a type parameter and resolved inside the generic
// class, where the imports of the source no longer apply, so it has to be
// spelled in full first.
func TestResolve_ATypeArgumentNamedByAnImportOnDemandIsQualified(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"com.example.lib.Wrapper": {
			Name: "Wrapper", Package: "com.example.lib", TypeParameters: []string{"T"},
			Fields: []parser.Field{{Name: "code", Type: "int"}, {Name: "data", Type: "T"}},
		},
		"com.example.lib.model.Dto": {Name: "Dto", Package: "com.example.lib.model", Fields: []parser.Field{{Name: "id", Type: "long"}}},
	}}
	controller := parser.Class{
		Name: "DtoController", Package: "com.example.app", FilePath: "DtoController.java",
		Imports: append(onDemand("com.example.lib.model"), parser.Import{Path: "com.example.lib.Wrapper"}),
	}
	r := NewTypeResolver([]parser.Class{controller})
	r.SetDependencyResolver(library)

	got := r.qualifyTypeArgs(controller, nil, []string{"Dto", "List<Dto>", "Map<String, Dto[]>"})
	want := []string{"com.example.lib.model.Dto", "List<com.example.lib.model.Dto>", "Map<java.lang.String, com.example.lib.model.Dto[]>"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("qualifyTypeArgs:\n got  %q\n want %q", got, want)
	}

	data := r.ResolveFrom(controller, "Wrapper<Dto>", nil).Fields["data"].Model
	if data.Kind != model.KindObject || data.Fields["id"] == nil {
		t.Errorf("data = %+v, want Dto expanded", data)
	}
	list := r.ResolveFrom(controller, "Wrapper<List<Dto>>", nil).Fields["data"].Model
	if list.Kind != model.KindArray || list.Items.Fields["id"] == nil {
		t.Errorf("data = %+v, want an array of Dto", list)
	}
}

// A source names the class its scope names: a library class it imports on demand
// is the one, and a class of the project that the source reaches neither by an
// import nor by its package does not take its place. A source that reaches no
// Level keeps falling back to the project's class, and the library class read for
// the other source does not answer for it. (A type argument spelled by its simple
// name inside a generic class is qualified before it travels, so the scope of
// that class is never consulted for it; see qualifyTypeName.)
func TestResolve_ALibraryClassFoundByAnImportOnDemandBeatsAProjectClassTheSourceDoesNotReach(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"com.example.lib.Level": {Name: "Level", Package: "com.example.lib", Fields: []parser.Field{{Name: "rank", Type: "int"}}},
	}}
	// Reaches the library's Level through an import on demand.
	imported := parser.Class{Name: "Imported", Package: "com.example.web", FilePath: "Imported.java", Imports: onDemand("com.example.lib")}
	// Imports nothing that declares a Level.
	bare := parser.Class{Name: "Bare", Package: "com.example.web", FilePath: "Bare.java"}
	r := NewTypeResolver([]parser.Class{
		{Name: "Level", Package: "com.example.web.model", FilePath: "Level.java", Fields: []parser.Field{{Name: "score", Type: "int"}}},
		imported, bare,
	})
	r.SetDependencyResolver(library)

	assertFieldNames(t, fieldNamesOf(t, r, imported, "Level"), []string{"rank"})
	assertFieldNames(t, fieldNamesOf(t, r, bare, "Level"), []string{"score"})
}

// A library class read through one source does not stand in for an enum of the
// project, which the JSON view does not see: the enum is exported by its name,
// as an opaque value.
func TestResolve_ALibraryClassDoesNotAnswerForAProjectEnumsName(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		// Not an enum, and sorts before the project's Level.
		"com.example.lib.Level": {Name: "Level", Package: "com.example.lib", Fields: []parser.Field{{Name: "rank", Type: "int"}}},
	}}
	// Reaches the library's Level and nothing but the library's.
	imported := parser.Class{Name: "Imported", Package: "com.example.web", FilePath: "Imported.java", Imports: onDemand("com.example.lib")}
	// Names the enum by nothing at all, and by an import on demand ahead of
	// packages that declare no Level.
	bare := parser.Class{Name: "Bare", Package: "com.example.web", FilePath: "Bare.java"}
	both := parser.Class{Name: "Both", Package: "com.example.web", FilePath: "Both.java", Imports: onDemand("com.example.web.enums", "com.example.other")}
	r := NewTypeResolver([]parser.Class{
		{Name: "Level", Package: "com.example.web.enums", FilePath: "Level.java", IsEnum: true, EnumConstants: []string{"LOW", "HIGH"}},
		imported, bare, both,
	})
	r.SetDependencyResolver(library)

	assertFieldNames(t, fieldNamesOf(t, r, imported, "Level"), []string{"rank"})
	for _, source := range []parser.Class{bare, both} {
		level := r.ResolveFrom(source, "Level", nil)
		if level.Kind != model.KindSingle || level.TypeName != "Level" {
			t.Errorf("Level from %s = %+v, want the enum's name", source.Name, level)
		}
	}
	for _, name := range library.asked {
		// Both's import of the enum's package ends the search before the
		// package after it is tried.
		if name == "com.example.other.Level" {
			t.Errorf("%s was asked for: %v", name, library.asked)
		}
	}
}

// Which class a name denotes does not depend on what the resolver has read
// before: the package of the source comes before an import on demand, and the
// imports on demand keep the order they are written in, whether the classes
// they would name are already held or still to be asked for (JLS 6.4.1).
func TestResolve_TheScopeOrderDoesNotDependOnWhatWasReadBefore(t *testing.T) {
	classes := map[string]parser.Class{
		"lib.a.Holder": {
			Name: "Holder", Package: "lib.a", Imports: onDemand("lib.b"),
			Fields: []parser.Field{{Name: "shared", Type: "Shared"}},
		},
		"lib.a.Shared": {Name: "Shared", Package: "lib.a", Fields: []parser.Field{{Name: "fromSamePackage", Type: "String"}}},
		"lib.b.Shared": {Name: "Shared", Package: "lib.b", Fields: []parser.Field{{Name: "fromOnDemand", Type: "String"}}},

		"lib.c.Many": {
			Name: "Many", Package: "lib.c", Imports: onDemand("lib.d", "lib.e"),
			Fields: []parser.Field{{Name: "thing", Type: "Thing"}},
		},
		"lib.d.Thing": {Name: "Thing", Package: "lib.d", Fields: []parser.Field{{Name: "fromFirst", Type: "String"}}},
		"lib.e.Thing": {Name: "Thing", Package: "lib.e", Fields: []parser.Field{{Name: "fromSecond", Type: "String"}}},
	}
	// A source that names the wildcard class through its own import on demand.
	reader := parser.Class{Name: "Reader", Package: "app", FilePath: "Reader.java", Imports: onDemand("lib.b")}
	for _, tc := range []struct {
		name   string
		holder string
		// preload reads a class before the holder is resolved; loaded is its key.
		preload func(r *TypeResolver)
		loaded  string
		want    string
	}{
		{name: "same package, nothing read before", holder: "lib.a.Holder", want: "fromSamePackage"},
		{
			name: "same package, the wildcard class read by its qualified name", holder: "lib.a.Holder", want: "fromSamePackage",
			preload: func(r *TypeResolver) { r.ResolveFrom(parser.Class{}, "lib.b.Shared", nil) }, loaded: "lib.b.Shared",
		},
		{
			name: "same package, the wildcard class read through an import", holder: "lib.a.Holder", want: "fromSamePackage",
			preload: func(r *TypeResolver) { r.ResolveFrom(reader, "Shared", nil) }, loaded: "lib.b.Shared",
		},
		{name: "first import, nothing read before", holder: "lib.c.Many", want: "fromFirst"},
		{
			name: "first import, the later import's class read before", holder: "lib.c.Many", want: "fromFirst",
			preload: func(r *TypeResolver) { r.ResolveFrom(parser.Class{}, "lib.e.Thing", nil) }, loaded: "lib.e.Thing",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			library := &libraryClasses{classes: classes}
			r := NewTypeResolver(nil)
			r.SetDependencyResolver(library)

			if tc.preload != nil {
				tc.preload(r)
				if _, held := r.classRegistry[tc.loaded]; !held {
					t.Fatalf("%s was not read before the holder is resolved", tc.loaded)
				}
			}

			holder := r.ResolveFrom(parser.Class{}, tc.holder, nil)
			var fieldModel *model.ObjectModel
			for _, field := range holder.Fields {
				fieldModel = field.Model
			}
			if fieldModel == nil || fieldModel.Kind != model.KindObject {
				t.Fatalf("the field of %s = %+v, want an expanded object", tc.holder, fieldModel)
			}
			var got []string
			for name := range fieldModel.Fields {
				got = append(got, name)
			}
			if !reflect.DeepEqual(got, []string{tc.want}) {
				t.Errorf("the field of %s has %v, want %q", tc.holder, got, tc.want)
			}
		})
	}
}

// The class a source reaches by an import on demand is the same before and after
// it has been read, and a source that does not reach it keeps the project's class
// either way: reading the library class moves neither ahead.
func TestResolve_ReadingALibraryClassMovesNeitherItNorTheProjectClassAhead(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"lib.Level": {Name: "Level", Package: "lib", Fields: []parser.Field{{Name: "rank", Type: "int"}}},
	}}
	imported := parser.Class{Name: "Imported", Package: "app.web", FilePath: "Imported.java", Imports: onDemand("lib")}
	bare := parser.Class{Name: "Bare", Package: "app.web", FilePath: "Bare.java"}
	r := NewTypeResolver([]parser.Class{
		{Name: "Level", Package: "app.model", FilePath: "Level.java", Fields: []parser.Field{{Name: "score", Type: "int"}}},
		imported, bare,
	})
	r.SetDependencyResolver(library)

	assertFieldNames(t, fieldNamesOf(t, r, imported, "Level"), []string{"rank"})
	assertFieldNames(t, fieldNamesOf(t, r, bare, "Level"), []string{"score"})
	// Read through its qualified name, as a source that spells it in full does.
	assertFieldNames(t, fieldNamesOf(t, r, parser.Class{}, "lib.Level"), []string{"rank"})
	assertFieldNames(t, fieldNamesOf(t, r, imported, "Level"), []string{"rank"})
	assertFieldNames(t, fieldNamesOf(t, r, bare, "Level"), []string{"score"})
	if class, found := r.Class(bare, "Level"); !found || class.Package != "app.model" {
		t.Errorf("Class(bare, Level) = %+v, %v, want the project's class", class, found)
	}
	if class, found := r.Class(imported, "Level"); !found || class.Package != "lib" {
		t.Errorf("Class(imported, Level) = %+v, %v, want the library's class", class, found)
	}
}

// A class read from a library names its own neighbours without an import, and
// an unrelated class of the project that carries the same simple name does not
// take their place: the names of a library class follow the scope of that class
// (a single-type import, its package, an import on demand), whatever the
// project declares and whatever the resolver has read before.
func TestResolve_ALibraryClassNamesItsOwnPackageBeforeAProjectClassOfTheSameName(t *testing.T) {
	classes := map[string]parser.Class{
		"lib.req.Request": {
			Name: "Request", Package: "lib.req",
			Imports: append([]parser.Import{{Path: "lib.file.FileInfo"}}, onDemand("lib.extra")...),
			Fields: []parser.Field{
				// Declared in the package of Request: no import names it.
				{Name: "tiers", Type: "List<TierItem>"},
				{Name: "file", Type: "FileInfo"},
				{Name: "extra", Type: "Extra"},
				// The package comes before an import on demand.
				{Name: "both", Type: "Both"},
			},
		},
		"lib.req.TierItem":  {Name: "TierItem", Package: "lib.req", Fields: []parser.Field{{Name: "fromLibrary", Type: "String"}}},
		"lib.file.FileInfo": {Name: "FileInfo", Package: "lib.file", Fields: []parser.Field{{Name: "fromLibrary", Type: "String"}}},
		"lib.extra.Extra":   {Name: "Extra", Package: "lib.extra", Fields: []parser.Field{{Name: "fromLibrary", Type: "String"}}},
		"lib.req.Both":      {Name: "Both", Package: "lib.req", Fields: []parser.Field{{Name: "fromPackage", Type: "String"}}},
		"lib.extra.Both":    {Name: "Both", Package: "lib.extra", Fields: []parser.Field{{Name: "fromOnDemand", Type: "String"}}},
	}
	// Classes of the project that are not the library's, under the simple names
	// the library class uses.
	project := func() []parser.Class {
		var out []parser.Class
		for _, name := range []string{"TierItem", "FileInfo", "Extra", "Both"} {
			out = append(out, parser.Class{
				Name: name, Package: "app.dto", FilePath: name + ".java",
				Fields: []parser.Field{{Name: "fromProject", Type: "String"}},
			})
		}
		return out
	}
	controller := parser.Class{Name: "Controller", Package: "app.web", FilePath: "Controller.java", Imports: onDemand("app.dto")}

	for _, tc := range []struct {
		name string
		// preload reads a class before the library class is expanded.
		preload func(r *TypeResolver)
	}{
		{name: "nothing read before"},
		{name: "the project's class read first", preload: func(r *TypeResolver) {
			for _, name := range []string{"TierItem", "FileInfo", "Extra", "Both"} {
				if got := fieldNamesOf(t, r, controller, name); !reflect.DeepEqual(got, []string{"fromProject"}) {
					t.Fatalf("%s from the controller has %v, want the project's", name, got)
				}
			}
		}},
		{name: "the library's neighbours read first", preload: func(r *TypeResolver) {
			for _, name := range []string{"lib.req.TierItem", "lib.file.FileInfo", "lib.extra.Extra", "lib.extra.Both"} {
				r.ResolveFrom(parser.Class{}, name, nil)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			library := &libraryClasses{classes: classes}
			r := NewTypeResolver(append(project(), controller))
			r.SetDependencyResolver(library)
			if tc.preload != nil {
				tc.preload(r)
			}

			request := r.ResolveFrom(parser.Class{}, "lib.req.Request", nil)
			namesOf := func(field string) []string {
				m := request.Fields[field].Model
				if m != nil && m.Kind == model.KindArray {
					m = m.Items
				}
				if m == nil || m.Kind != model.KindObject {
					t.Fatalf("%s = %+v, want an expanded object", field, m)
				}
				var names []string
				for name := range m.Fields {
					names = append(names, name)
				}
				return names
			}
			for field, want := range map[string]string{
				"tiers": "fromLibrary", "file": "fromLibrary", "extra": "fromLibrary", "both": "fromPackage",
			} {
				if got := namesOf(field); !reflect.DeepEqual(got, []string{want}) {
					t.Errorf("%s of the library class has %v, want [%s]", field, got, want)
				}
			}
		})
	}
}

// A name that a source of the project wrote and a library class receives, as a
// type argument bound to a type parameter or as the argument of a superclass,
// is the project's class, not the same-named class next to the library class.
// The names of the library class itself follow its own scope (see
// TestResolve_ALibraryClassNamesItsOwnPackageBeforeAProjectClassOfTheSameName).
func TestResolve_AProjectClassPassedToALibraryClassKeepsItsMeaning(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"lib.Page": {
			Name: "Page", Package: "lib", TypeParameters: []string{"T"},
			Fields: []parser.Field{
				{Name: "first", Type: "T"},
				{Name: "rows", Type: "List<T>"},
				{Name: "byKey", Type: "Map<String, T>"},
				{Name: "wide", Type: "List<? extends T>"},
				// Written in the library: its own Item.
				{Name: "own", Type: "Item"},
			},
		},
		"lib.Base": {
			Name: "Base", Package: "lib", TypeParameters: []string{"T"},
			Fields: []parser.Field{{Name: "payload", Type: "T"}, {Name: "ownItem", Type: "Item"}},
		},
		// Not the class of the project.
		"lib.Item": {Name: "Item", Package: "lib", Fields: []parser.Field{{Name: "fromLibrary", Type: "String"}}},
	}}
	controller := parser.Class{Name: "Controller", Package: "app.web", FilePath: "Controller.java", Imports: []parser.Import{{Path: "app.dto.Item"}, {Path: "lib.Page"}}}
	derived := parser.Class{
		Name: "Derived", Package: "app.web", FilePath: "Derived.java",
		Imports:    []parser.Import{{Path: "app.dto.Item"}, {Path: "lib.Base"}},
		SuperClass: "Base", SuperClassTypeArgs: []string{"Item"},
	}
	r := NewTypeResolver([]parser.Class{
		{Name: "Item", Package: "app.dto", FilePath: "Item.java", Fields: []parser.Field{{Name: "fromProject", Type: "String"}}},
		controller, derived,
	})
	r.SetDependencyResolver(library)
	// The superclass is expanded when the resolver holds it.
	r.ResolveFrom(parser.Class{}, "lib.Base", nil)

	namesOf := func(m *model.ObjectModel) []string {
		for m != nil && (m.Kind == model.KindArray || m.Kind == model.KindMap) {
			if m.Kind == model.KindArray {
				m = m.Items
			} else {
				m = m.ValueModel
			}
		}
		if m == nil || m.Kind != model.KindObject {
			t.Fatalf("model = %+v, want an expanded object", m)
		}
		var names []string
		for name := range m.Fields {
			names = append(names, name)
		}
		return names
	}

	page := r.ResolveFrom(controller, "Page<Item>", nil)
	for field, want := range map[string]string{"first": "fromProject", "rows": "fromProject", "byKey": "fromProject", "wide": "fromProject", "own": "fromLibrary"} {
		if got := namesOf(page.Fields[field].Model); !reflect.DeepEqual(got, []string{want}) {
			t.Errorf("Page<Item>.%s has %v, want [%s]", field, got, want)
		}
	}

	extended := r.ResolveFrom(controller, "Derived", nil)
	for field, want := range map[string]string{"payload": "fromProject", "ownItem": "fromLibrary"} {
		if extended.Fields[field] == nil {
			t.Fatalf("Derived = %+v, want the field %s of its superclass", extended, field)
		}
		if got := namesOf(extended.Fields[field].Model); !reflect.DeepEqual(got, []string{want}) {
			t.Errorf("Derived.%s has %v, want [%s]", field, got, want)
		}
	}
}

// A class of the project that a library generic class receives is named by its
// qualified name wherever the library class reads it again, as it does for the
// element of a list, a map value or a bounded wildcard. The library is not asked
// for it by the package of the library class, since each of those lookups is a
// call to the indexer, and another class of the project that carries the same
// simple name does not stand in for it.
func TestResolve_AProjectClassPassedToALibraryClassIsNotAskedOfTheLibrary(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"lib.Page": {
			Name: "Page", Package: "lib", TypeParameters: []string{"T"},
			Fields: []parser.Field{
				{Name: "first", Type: "T"},
				{Name: "rows", Type: "List<T>"},
				{Name: "byKey", Type: "Map<String, T>"},
				{Name: "wide", Type: "List<? extends T>"},
			},
		},
	}}
	controller := parser.Class{Name: "Controller", Package: "app.web", FilePath: "Controller.java", Imports: []parser.Import{{Path: "lib.Page"}, {Path: "app.dto.Item"}}}
	r := NewTypeResolver([]parser.Class{
		{Name: "Item", Package: "app.dto", FilePath: "Item.java", Fields: []parser.Field{{Name: "fromProject", Type: "String"}}},
		// Another Item of the project, which sorts first and which controller
		// does not import.
		{Name: "Item", Package: "app.api", FilePath: "api/Item.java", Fields: []parser.Field{{Name: "fromOther", Type: "String"}}},
		controller,
	})
	r.SetDependencyResolver(library)

	page := r.ResolveFrom(controller, "Page<Item>", nil)
	if page.Kind != model.KindObject || len(page.Fields) != 4 {
		t.Fatalf("Page<Item> = %+v, want Page with its four fields", page)
	}
	for name, field := range page.Fields {
		m := field.Model
		for m != nil && (m.Kind == model.KindArray || m.Kind == model.KindMap) {
			if m.Kind == model.KindArray {
				m = m.Items
			} else {
				m = m.ValueModel
			}
		}
		if m == nil || m.Kind != model.KindObject || m.Fields["fromProject"] == nil {
			t.Errorf("%s = %+v, want the project's Item", name, field.Model)
		}
	}
	if want := []string{"lib.Page"}; !reflect.DeepEqual(library.asked, want) {
		t.Errorf("asked = %v, want %v", library.asked, want)
	}
}

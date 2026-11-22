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

func TestResolve_NoLookupForAPlatformPackageAClassImportOrAKnownName(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{}}
	// Read from a library, so its own package may be asked.
	source := parser.Class{
		Name: "Holder", Package: "com.example.app",
		Imports: append(onDemand("java.util", "javax.validation.constraints", "kotlin.collections", "com.example.lib.Outer"),
			parser.Import{Path: "com.example.lib.Statics", OnDemand: true, Static: true}),
	}
	// T is a type variable of some class; every other name is one the resolver
	// answers for itself.
	generic := parser.Class{Name: "Box", Package: "com.example.box", FilePath: "Box.java", TypeParameters: []string{"T"}}
	r := NewTypeResolver([]parser.Class{generic})
	r.SetDependencyResolver(library)

	for _, name := range []string{"T", "String", "Integer", "Object", "Date", "LocalDateTime", "UUID", "List", "Map"} {
		r.ResolveFrom(source, name, nil)
	}
	// Only the package of the source is left to ask, and then only for a name
	// that may be a class: the platform packages, the class import and the
	// static import give no candidate.
	r.ResolveFrom(source, "Unknown", nil)
	if want := []string{"com.example.app.Unknown", "Unknown"}; !reflect.DeepEqual(library.asked, want) {
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

// The project's classes win over the library's. The resolver spells a type by
// its simple name again when it expands a type argument inside a generic class,
// so a library class that answers to a simple name would take the place of a
// project class of that name, and of a class another source reaches under the
// same simple name (see TestSeeOptions_ALibraryClassDoesNotShadowAProjectEnum).
func TestResolve_ALibraryClassFoundByAnImportOnDemandDoesNotShadowAProjectClass(t *testing.T) {
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

	// The project class is the one the resolver has always answered with when
	// nothing in scope names the class: an import on demand does not displace it.
	assertFieldNames(t, fieldNamesOf(t, r, imported, "Level"), []string{"score"})
	assertFieldNames(t, fieldNamesOf(t, r, bare, "Level"), []string{"score"})
	if len(library.asked) != 0 {
		t.Errorf("the library was asked for %v, though the project declares Level", library.asked)
	}
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

// A class of the project that carries the name wins over a library class that
// the source reaches by its package or an import, and that holds also once the
// library class has been read: reading it does not move it ahead.
func TestResolve_AProjectClassKeepsWinningOnceTheLibraryClassIsRead(t *testing.T) {
	library := &libraryClasses{classes: map[string]parser.Class{
		"lib.Level": {Name: "Level", Package: "lib", Fields: []parser.Field{{Name: "rank", Type: "int"}}},
	}}
	imported := parser.Class{Name: "Imported", Package: "app.web", FilePath: "Imported.java", Imports: onDemand("lib")}
	r := NewTypeResolver([]parser.Class{
		{Name: "Level", Package: "app.model", FilePath: "Level.java", Fields: []parser.Field{{Name: "score", Type: "int"}}},
		imported,
	})
	r.SetDependencyResolver(library)

	assertFieldNames(t, fieldNamesOf(t, r, imported, "Level"), []string{"score"})
	// Read through its qualified name, as a source that spells it in full does.
	assertFieldNames(t, fieldNamesOf(t, r, parser.Class{}, "lib.Level"), []string{"rank"})
	assertFieldNames(t, fieldNamesOf(t, r, imported, "Level"), []string{"score"})
	if class, found := r.Class(imported, "Level"); !found || class.Package != "app.model" {
		t.Errorf("Class(Level) = %+v, %v, want the project's class", class, found)
	}
}

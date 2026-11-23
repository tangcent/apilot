package resolver

import (
	"reflect"
	"sort"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// A declaration of one of the three kinds the lookup meets: a class of the
// scanned sources, a class already read from the dependencies, and a class of the
// dependencies that has not been read yet.
type declKind string

const (
	declScanned declKind = "scanned"
	declRead    declKind = "read"
	declAsked   declKind = "asked"
)

var allDeclKinds = []declKind{declScanned, declRead, declAsked}

// scopeWorld places classes by kind: scanned ones go to the resolver's sources,
// the others to a library, which a preload step reads for the "read" kind.
type scopeWorld struct {
	scanned []parser.Class
	library *libraryClasses
	read    []string
}

func newScopeWorld() *scopeWorld {
	return &scopeWorld{library: &libraryClasses{classes: map[string]parser.Class{}}}
}

// place adds class under the given kind. The marker is the field that tells
// which class a name resolved to.
func (w *scopeWorld) place(kind declKind, class parser.Class) {
	switch kind {
	case declScanned:
		if class.FilePath == "" {
			class.FilePath = class.Name + "-" + class.Package + ".java"
		}
		w.scanned = append(w.scanned, class)
	case declRead:
		w.library.classes[qualifiedName(class)] = class
		w.read = append(w.read, qualifiedName(class))
	default:
		w.library.classes[qualifiedName(class)] = class
	}
}

// resolver builds the resolver and reads the classes placed as "read" through
// their qualified names, the way a source that spells them in full does.
func (w *scopeWorld) resolver(sources ...parser.Class) *TypeResolver {
	r := NewTypeResolver(append(append([]parser.Class{}, w.scanned...), sources...))
	r.SetDependencyResolver(w.library)
	for _, key := range w.read {
		r.ResolveFrom(parser.Class{}, key, nil)
	}
	return r
}

// marked is a class of the given package whose only field is marker.
func marked(pkg, name, marker string) parser.Class {
	return parser.Class{Name: name, Package: pkg, Fields: []parser.Field{{Name: marker, Type: "String"}}}
}

func fieldsOf(m *model.ObjectModel) []string {
	if m == nil || m.Kind != model.KindObject {
		return nil
	}
	var names []string
	for name := range m.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// The scope of a name is walked tier by tier (JLS 6.4.1), and a tier answers
// whatever kind of declaration it holds: the package of the source beats an
// import on demand when the one is a class read from a library and the other a
// class of the scanned sources, and in every other pairing. The class a name
// denotes does not depend on where its declaration comes from, on whether it was
// read before, or on whether the source is a library class or a source of the
// project.
func TestResolve_ATierAnswersBeforeTheNextWhateverKindOfDeclarationItHolds(t *testing.T) {
	for _, packageKind := range allDeclKinds {
		for _, wildcardKind := range allDeclKinds {
			t.Run("library source, package "+string(packageKind)+", wildcard "+string(wildcardKind), func(t *testing.T) {
				w := newScopeWorld()
				w.place(packageKind, marked("lib.a", "Shared", "fromPackage"))
				w.place(wildcardKind, marked("lib.b", "Shared", "fromWildcard"))
				// A class read from a library: not a source of the project.
				holder := parser.Class{Name: "Holder", Package: "lib.a", Imports: onDemand("lib.b"), Fields: []parser.Field{{Name: "shared", Type: "Shared"}}}
				w.library.classes["lib.a.Holder"] = holder
				r := w.resolver()

				got := fieldsOf(r.ResolveFrom(parser.Class{}, "lib.a.Holder", nil).Fields["shared"].Model)
				if !reflect.DeepEqual(got, []string{"fromPackage"}) {
					t.Errorf("Shared has %v, want the class of the package of Holder", got)
				}
			})

			// A package of the project holds the project's classes only (see
			// TestResolve_ASourceOfTheProjectDoesNotSeeADependencyClassInAPackageItDeclares).
			if packageKind != declScanned {
				continue
			}
			t.Run("project source, package "+string(packageKind)+", wildcard "+string(wildcardKind), func(t *testing.T) {
				w := newScopeWorld()
				w.place(packageKind, marked("app.web", "Shared", "fromPackage"))
				w.place(wildcardKind, marked("lib.b", "Shared", "fromWildcard"))
				controller := parser.Class{Name: "Controller", Package: "app.web", FilePath: "Controller.java", Imports: onDemand("lib.b")}
				r := w.resolver(controller)

				got := fieldsOf(r.ResolveFrom(controller, "Shared", nil))
				if !reflect.DeepEqual(got, []string{"fromPackage"}) {
					t.Errorf("Shared has %v, want the class of the package of Controller", got)
				}
			})
		}
	}
}

// A package that the project declares holds the project's classes for a source of
// the project, and a class of the dependencies in it is not seen by its simple
// name, whether or not it has been read: the lookups the project's packages would
// cost are the price of completing the same-package tier (see declares). The
// source can still spell the class in full, and a class read from a library finds
// its own neighbours there.
func TestResolve_ASourceOfTheProjectDoesNotSeeADependencyClassInAPackageItDeclares(t *testing.T) {
	for _, kind := range []declKind{declRead, declAsked} {
		t.Run(string(kind), func(t *testing.T) {
			w := newScopeWorld()
			// The package app.web is the project's: Controller is declared in it.
			w.place(kind, marked("app.web", "Shared", "fromDependency"))
			w.place(kind, marked("lib.b", "Shared", "fromWildcard"))
			controller := parser.Class{Name: "Controller", Package: "app.web", FilePath: "Controller.java", Imports: onDemand("lib.b")}
			r := w.resolver(controller)

			if got := fieldsOf(r.ResolveFrom(controller, "Shared", nil)); !reflect.DeepEqual(got, []string{"fromWildcard"}) {
				t.Errorf("Shared from the project has %v, want the class the import on demand names", got)
			}
			if got := fieldsOf(r.ResolveFrom(controller, "app.web.Shared", nil)); !reflect.DeepEqual(got, []string{"fromDependency"}) {
				t.Errorf("app.web.Shared has %v, want the dependency's class by its qualified name", got)
			}
			neighbour := parser.Class{Name: "Neighbour", Package: "app.web", Imports: onDemand("lib.b")}
			if got := fieldsOf(r.ResolveFrom(neighbour, "Shared", nil)); !reflect.DeepEqual(got, []string{"fromDependency"}) {
				t.Errorf("Shared from a class of a library in app.web has %v, want its neighbour", got)
			}
		})
	}
}

// The imports on demand keep the order they are written in, whichever kind of
// declaration each package holds.
func TestResolve_ImportsOnDemandKeepTheirOrderAcrossKindsOfDeclaration(t *testing.T) {
	for _, firstKind := range allDeclKinds {
		for _, secondKind := range allDeclKinds {
			t.Run("first "+string(firstKind)+", second "+string(secondKind), func(t *testing.T) {
				w := newScopeWorld()
				w.place(firstKind, marked("lib.d", "Thing", "fromFirst"))
				w.place(secondKind, marked("lib.e", "Thing", "fromSecond"))
				controller := parser.Class{Name: "Controller", Package: "app.web", FilePath: "Controller.java", Imports: onDemand("lib.d", "lib.e")}
				r := w.resolver(controller)

				if got := fieldsOf(r.ResolveFrom(controller, "Thing", nil)); !reflect.DeepEqual(got, []string{"fromFirst"}) {
					t.Errorf("Thing has %v, want the class of the first import", got)
				}
			})
		}
	}
}

// A single-type import names its class whatever else the package and the other
// imports declare, and a class declared in the file beats the import.
func TestResolve_ASingleTypeImportComesBeforeThePackageAndTheImportsOnDemand(t *testing.T) {
	for _, otherKind := range allDeclKinds {
		t.Run(string(otherKind), func(t *testing.T) {
			w := newScopeWorld()
			w.place(declAsked, marked("lib.i", "Shared", "fromImport"))
			w.place(otherKind, marked("app.web", "Shared", "fromPackage"))
			w.place(otherKind, marked("lib.b", "Shared", "fromWildcard"))
			controller := parser.Class{
				Name: "Controller", Package: "app.web", FilePath: "Controller.java",
				Imports: append([]parser.Import{{Path: "lib.i.Shared"}}, onDemand("lib.b")...),
			}
			r := w.resolver(controller)

			if got := fieldsOf(r.ResolveFrom(controller, "Shared", nil)); !reflect.DeepEqual(got, []string{"fromImport"}) {
				t.Errorf("Shared has %v, want the imported class", got)
			}
		})
	}
}

// The declaration of the project that carries a name anywhere answers only after
// every tier of the scope has: a class the source reaches through an import on
// demand is the class it names, and a project class it cannot reach does not take
// its place, whether or not the library class has been read.
func TestResolve_AProjectClassElsewhereAnswersAfterEveryTier(t *testing.T) {
	for _, kind := range []declKind{declRead, declAsked} {
		t.Run(string(kind), func(t *testing.T) {
			w := newScopeWorld()
			w.place(kind, marked("lib", "Status", "fromLibrary"))
			w.place(declScanned, marked("other.model", "Status", "fromProject"))
			controller := parser.Class{Name: "Controller", Package: "app.web", FilePath: "Controller.java", Imports: onDemand("lib")}
			stranger := parser.Class{Name: "Stranger", Package: "app.web", FilePath: "Stranger.java"}
			r := w.resolver(controller, stranger)

			if got := fieldsOf(r.ResolveFrom(controller, "Status", nil)); !reflect.DeepEqual(got, []string{"fromLibrary"}) {
				t.Errorf("Status from a source that imports lib.* has %v, want the library's", got)
			}
			// A source that reaches neither falls back to the project's class, and
			// the library class that was read does not answer for the name.
			if got := fieldsOf(r.ResolveFrom(stranger, "Status", nil)); !reflect.DeepEqual(got, []string{"fromProject"}) {
				t.Errorf("Status from a source that imports nothing has %v, want the project's", got)
			}
		})
	}
}

// The same holds for a class read from a library: a project class it cannot
// reach does not take the place of a class its own scope names.
func TestResolve_ALibraryClassFindsItsInScopeNeighbourBeforeAProjectClassElsewhere(t *testing.T) {
	for _, kind := range allDeclKinds {
		t.Run(string(kind), func(t *testing.T) {
			w := newScopeWorld()
			w.place(kind, marked("lib", "Status", "fromLibrary"))
			w.place(declScanned, marked("app.model", "Status", "fromProject"))
			holder := parser.Class{Name: "Holder", Package: "lib", Fields: []parser.Field{{Name: "status", Type: "Status"}}}
			w.library.classes["lib.Holder"] = holder
			r := w.resolver()

			if got := fieldsOf(r.ResolveFrom(parser.Class{}, "lib.Holder", nil).Fields["status"].Model); !reflect.DeepEqual(got, []string{"fromLibrary"}) {
				t.Errorf("Status has %v, want the class of the package of Holder", got)
			}
		})
	}
}

// A dependency class that names a class by its qualified name keeps that class
// when a class of the same simple name sits next to it (the library's own Item
// against the project's app.dto.Item), through a collection, a generic class of
// the library and a generic class nested in a list, with the other class read
// before or not.
func TestResolve_ALibraryClassKeepsTheQualifiedClassItNames(t *testing.T) {
	for _, itemKind := range []declKind{declScanned, declRead, declAsked} {
		for _, warm := range []string{"cold", "library Item read first", "app.dto.Item read first"} {
			t.Run("app.dto.Item "+string(itemKind)+", "+warm, func(t *testing.T) {
				w := newScopeWorld()
				w.place(itemKind, marked("app.dto", "Item", "fromApp"))
				w.library.classes["lib.Item"] = marked("lib", "Item", "fromLibrary")
				w.library.classes["lib.Box"] = parser.Class{
					Name: "Box", Package: "lib", TypeParameters: []string{"T"},
					Fields: []parser.Field{{Name: "value", Type: "T"}, {Name: "all", Type: "List<T>"}},
				}
				w.library.classes["lib.Holder"] = parser.Class{
					Name: "Holder", Package: "lib",
					Fields: []parser.Field{
						{Name: "items", Type: "List<app.dto.Item>"},
						{Name: "box", Type: "Box<app.dto.Item>"},
						{Name: "boxes", Type: "List<Box<app.dto.Item>>"},
						{Name: "byKey", Type: "Map<String, app.dto.Item>"},
						{Name: "wide", Type: "Box<? extends app.dto.Item>"},
						{Name: "plain", Type: "app.dto.Item"},
						{Name: "own", Type: "Item"},
					},
				}
				r := w.resolver()
				switch warm {
				case "library Item read first":
					r.ResolveFrom(parser.Class{}, "lib.Item", nil)
				case "app.dto.Item read first":
					r.ResolveFrom(parser.Class{}, "app.dto.Item", nil)
				}

				holder := r.ResolveFrom(parser.Class{}, "lib.Holder", nil)
				leaf := func(m *model.ObjectModel) *model.ObjectModel {
					for m != nil && (m.Kind == model.KindArray || m.Kind == model.KindMap) {
						if m.Kind == model.KindArray {
							m = m.Items
						} else {
							m = m.ValueModel
						}
					}
					return m
				}
				for field, want := range map[string]string{
					"items": "fromApp", "plain": "fromApp", "byKey": "fromApp", "own": "fromLibrary",
				} {
					if got := fieldsOf(leaf(holder.Fields[field].Model)); !reflect.DeepEqual(got, []string{want}) {
						t.Errorf("%s has %v, want [%s]", field, got, want)
					}
				}
				for _, field := range []string{"box", "wide"} {
					box := holder.Fields[field].Model
					for _, inner := range []string{"value", "all"} {
						if got := fieldsOf(leaf(box.Fields[inner].Model)); !reflect.DeepEqual(got, []string{"fromApp"}) {
							t.Errorf("%s.%s has %v, want [fromApp]", field, inner, got)
						}
					}
				}
				boxes := leaf(holder.Fields["boxes"].Model)
				for _, inner := range []string{"value", "all"} {
					if got := fieldsOf(leaf(boxes.Fields[inner].Model)); !reflect.DeepEqual(got, []string{"fromApp"}) {
						t.Errorf("boxes.%s has %v, want [fromApp]", inner, got)
					}
				}
			})
		}
	}
}

// A dependency enum that a source reaches through an import on demand is the
// class its name denotes, before a project class of the same name that the source
// cannot reach, whether or not it was read before. The JSON view does not see an
// enum, so the name is exported as an opaque value; the binding view takes it as a
// string. Either way the enum is held as a class of the dependencies (see
// registerDependency), so it never stands in for a class of the project.
func TestResolve_ADependencyEnumReachedByAnImportOnDemandAnswersBeforeAProjectClassElsewhere(t *testing.T) {
	for _, warm := range []bool{false, true} {
		name := "fresh"
		if warm {
			name = "read before"
		}
		t.Run(name, func(t *testing.T) {
			library := &libraryClasses{classes: map[string]parser.Class{
				"lib.Status": {Name: "Status", Package: "lib", IsEnum: true, EnumConstants: []string{"ON", "OFF"}},
			}}
			controller := parser.Class{Name: "Controller", Package: "app.web", FilePath: "Controller.java", Imports: onDemand("lib")}
			r := NewTypeResolver([]parser.Class{controller, marked("other.model", "Status", "fromProject")})
			r.SetDependencyResolver(library)
			if warm {
				r.ResolveForBinding(parser.Class{}, "lib.Status", nil)
			}

			if got := r.ResolveFrom(controller, "Status", nil); got.Kind != model.KindSingle {
				t.Errorf("Status in the JSON view = %+v, want the enum's opaque value", got)
			}
			if got := r.ResolveForBinding(controller, "Status", nil); got.Kind != model.KindSingle || got.TypeName != model.JsonTypeString {
				t.Errorf("Status in the binding view = %+v, want a string", got)
			}
			if !r.dependencyKeys["lib.Status"] {
				t.Errorf("lib.Status was not registered as a class of the dependencies: %v", r.dependencyKeys)
			}
			if got := r.bySimpleName["Status"]; !reflect.DeepEqual(got, []string{"other.model.Status"}) {
				t.Errorf("the scanned classes named Status = %v, want [other.model.Status]", got)
			}
		})
	}
}

// A class read from a library answers through a tier of the scope only, never by
// its simple name alone: a source that cannot reach it does not get it because
// another source read it first, so a name nothing in scope declares is the same
// opaque value before and after the read.
func TestResolve_ALibraryClassReadForOneSourceDoesNotAnswerForASourceThatCannotReachIt(t *testing.T) {
	w := newScopeWorld()
	w.place(declAsked, marked("lib", "Level", "fromLibrary"))
	imported := parser.Class{Name: "Imported", Package: "app.web", FilePath: "Imported.java", Imports: onDemand("lib")}
	stranger := parser.Class{Name: "Stranger", Package: "app.other", FilePath: "Stranger.java"}
	r := w.resolver(imported, stranger)

	if got := r.ResolveFrom(stranger, "Level", nil); got.Kind != model.KindSingle {
		t.Fatalf("Level from a source that reaches no Level, before the read = %+v, want an opaque value", got)
	}
	if got := fieldsOf(r.ResolveFrom(imported, "Level", nil)); !reflect.DeepEqual(got, []string{"fromLibrary"}) {
		t.Fatalf("Level from the source that imports lib.* has %v, want the library's", got)
	}
	if got := r.ResolveFrom(stranger, "Level", nil); got.Kind != model.KindSingle {
		t.Errorf("Level from a source that reaches no Level, after the read = %+v, want an opaque value", got)
	}
}

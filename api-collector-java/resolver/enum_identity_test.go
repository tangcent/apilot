package resolver

import (
	"fmt"
	"reflect"
	"sort"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// The tests below read real sources, so that a declaration reaches the resolver
// the way the parser hands it over, with the classes it is nested in. Each file
// is a compilation unit of its own, named by its path.

// parseFiles parses every source as the compilation unit of that path, in the
// order of the names, and returns the declarations of all of them.
func parseFiles(t *testing.T, files map[string]string) []parser.Class {
	t.Helper()
	jp, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer jp.Close()

	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	var all []parser.Class
	for _, path := range paths {
		classes, err := jp.ParseSource([]byte(files[path]))
		if err != nil {
			t.Fatal(err)
		}
		for i := range classes {
			classes[i].FilePath = path
		}
		all = append(all, classes...)
	}
	return all
}

// reversed returns the declarations in the opposite order, which is the order
// another file system listing would register them in.
func reversed(classes []parser.Class) []parser.Class {
	out := make([]parser.Class, len(classes))
	for i, c := range classes {
		out[len(classes)-1-i] = c
	}
	return out
}

// enumField is what a field of the exported schema says about the enum it
// names: the constants it lists, or none when the field is left unresolved.
type enumField struct {
	options    []any
	asString   bool // the model is the JSON string scalar
	unresolved bool // the name is reported as a type the resolver could not expand
}

// fieldOf resolves holder in the JSON view, or in the binding view, and reads
// one field of it. A collection or an array is read through to its items.
func fieldOf(classes []parser.Class, holder, name string, binding bool) enumField {
	r := NewTypeResolver(classes)
	r.SetUnresolved(collector.NewUnresolvedSet())
	return fieldFrom(r, holder, name, binding)
}

func fieldFrom(r *TypeResolver, holder, name string, binding bool) enumField {
	var resolved *model.ObjectModel
	if binding {
		resolved = r.ResolveForBinding(parser.Class{}, holder, nil)
	} else {
		resolved = r.Resolve(holder, nil)
	}
	field := resolved.Fields[name]
	if field == nil {
		return enumField{}
	}
	item := field.Model
	for item.Kind == model.KindArray || item.Kind == model.KindMap || (item.Kind == model.KindObject && item.Fields["data"] != nil) {
		switch {
		case item.Kind == model.KindArray:
			item = item.Items
		case item.Kind == model.KindMap:
			item = item.ValueModel
		default:
			item = item.Fields["data"].Model
		}
	}
	options := field.Options
	if data := field.Model.Fields["data"]; data != nil {
		options = data.Options
	}
	return enumField{
		options:    optionValues(options),
		asString:   item.Kind == model.KindSingle && item.TypeName == model.JsonTypeString,
		unresolved: item.Kind == model.KindSingle && item.TypeName != model.JsonTypeString,
	}
}

func (f enumField) String() string {
	switch {
	case f.unresolved:
		return fmt.Sprintf("unresolved, options %v", f.options)
	case f.asString:
		return fmt.Sprintf("string, options %v", f.options)
	}
	return fmt.Sprintf("other, options %v", f.options)
}

func wantUnresolved(f enumField) bool { return f.unresolved && len(f.options) == 0 }

func wantConstants(f enumField, constants ...string) bool {
	want := make([]any, len(constants))
	for i, c := range constants {
		want[i] = c
	}
	return f.asString && reflect.DeepEqual(f.options, want)
}

// Two enums nested in different classes of one file carry one simple name, and
// both declare the constant A. Only Order.Status declares @JsonValue, so the JSON
// view leaves it unresolved; Product.Status is no less itself for sharing a file,
// a simple name and its constants with it.
func TestResolve_NestedEnumsOfOneNameInAFileAreDifferentDeclarations(t *testing.T) {
	const declarations = `package p;

import com.fasterxml.jackson.annotation.JsonValue;

class Order {
    enum Status {
        A;
        @JsonValue String code() { return null; }
    }
}

class Product {
    enum Status { A }
}
`
	const holder = `package p;

import java.util.List;
import java.util.Map;

class Holder {
    Order.Status one;
    List<Order.Status> many;
    Order.Status[] array;
    Map<String, Order.Status> byName;
    Envelope<Order.Status> boxed;
    Envelope<List<Order.Status>> boxedMany;
    Product.Status productOne;
    List<Product.Status> productMany;
    Product.Status[] productArray;
    Envelope<Product.Status> productBoxed;
}

class Envelope<T> { T data; }
`
	classes := parseFiles(t, map[string]string{"p/Declarations.java": declarations, "p/Holder.java": holder})

	for order, ordered := range map[string][]parser.Class{"as declared": classes, "reversed": reversed(classes)} {
		for _, name := range []string{"one", "many", "array", "byName", "boxed", "boxedMany"} {
			if got := fieldOf(ordered, "p.Holder", name, false); !wantUnresolved(got) {
				t.Errorf("%s: Order.Status field %s = %v, want it unresolved: Order.Status declares @JsonValue", order, name, got)
			}
		}
		for _, name := range []string{"productOne", "productMany", "productArray", "productBoxed"} {
			if got := fieldOf(ordered, "p.Holder", name, false); !wantConstants(got, "A") {
				t.Errorf("%s: Product.Status field %s = %v, want a string of A", order, name, got)
			}
		}
	}
}

// A bare name written inside a class is the enum nested in that class before any
// other of the package or the file (JLS 6.4.1), in the JSON view and in the
// binding view alike, and whatever the order the declarations are registered in.
// A top-level enum of the same name is the one a class that declares no such
// member sees.
func TestResolve_ABareNameIsTheEnumNestedInTheClassThatWritesIt(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"p/Declarations.java": `package p;

import java.util.List;

class Order {
    enum Status { OPEN, CLOSED }
    Status own;
    List<Status> ownMany;
    Status[] ownArray;
    static class Line {
        Status inLine;
    }
}

class Product {
    enum Status { PHYSICAL, VIRTUAL }
    Status own;
    List<Status> ownMany;
    Status[] ownArray;
}

class Plain {
    Status top;
}
`,
		"p/Status.java": `package p;

public enum Status { TOP }
`,
	})

	for order, ordered := range map[string][]parser.Class{"as declared": classes, "reversed": reversed(classes)} {
		for _, binding := range []bool{false, true} {
			view := map[bool]string{false: "JSON", true: "binding"}[binding]
			for holder, want := range map[string][]string{
				"p.Order":   {"OPEN", "CLOSED"},
				"p.Product": {"PHYSICAL", "VIRTUAL"},
			} {
				names := []string{"own"}
				if !binding {
					names = append(names, "ownMany", "ownArray")
				}
				for _, name := range names {
					if got := fieldOf(ordered, holder, name, binding); !wantConstants(got, want...) {
						t.Errorf("%s, %s view: %s.%s = %v, want a string of %v", order, view, holder, name, got, want)
					}
				}
			}
			if got := fieldOf(ordered, "p.Plain", "top", binding); !wantConstants(got, "TOP") {
				t.Errorf("%s, %s view: Plain.top = %v, want a string of TOP: it declares no member Status", order, view, got)
			}
			if got := fieldOf(ordered, "p.Order.Line", "inLine", binding); !wantConstants(got, "OPEN", "CLOSED") {
				t.Errorf("%s, %s view: Order.Line.inLine = %v, want the Status of the class around it", order, view, got)
			}
		}
	}
}

// An enum nested in a class is named through that class, and a top-level enum of
// the same name, or a class nested in another class that carries it, is another
// declaration: Order.Status is the enum nested in Order wherever the others sit.
func TestResolve_ANestedEnumIsNotTakenForAnotherDeclarationOfItsName(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"p/Declarations.java": `package p;

class Order {
    enum Status { OPEN, CLOSED }
}

class Product {
    // A class of the same name, not an enum.
    static class Status { int weight; }
}

class Box {
    enum Status { FULL, EMPTY }
}
`,
		"p/Status.java": `package p;

public enum Status { TOP }
`,
		"p/Holder.java": `package p;

import java.util.List;
import java.util.Map;

class Holder {
    Status top;
    List<Status> tops;
    Order.Status order;
    List<Order.Status> orders;
    Map<String, Order.Status> byName;
    Box.Status box;
    Product.Status product;
}
`,
	})

	for order, ordered := range map[string][]parser.Class{"as declared": classes, "reversed": reversed(classes)} {
		for _, binding := range []bool{false, true} {
			view := map[bool]string{false: "JSON", true: "binding"}[binding]
			for name, want := range map[string][]string{"top": {"TOP"}, "order": {"OPEN", "CLOSED"}, "box": {"FULL", "EMPTY"}} {
				if got := fieldOf(ordered, "p.Holder", name, binding); !wantConstants(got, want...) {
					t.Errorf("%s, %s view: Holder.%s = %v, want a string of %v", order, view, name, got, want)
				}
			}
		}
		for name, want := range map[string][]string{"tops": {"TOP"}, "orders": {"OPEN", "CLOSED"}} {
			if got := fieldOf(ordered, "p.Holder", name, false); !wantConstants(got, want...) {
				t.Errorf("%s: Holder.%s = %v, want a string of %v", order, name, got, want)
			}
		}
		// The values of a map are strings of the constants, and the map lists none.
		if got := fieldOf(ordered, "p.Holder", "byName", false); !got.asString || len(got.options) != 0 {
			t.Errorf("%s: Holder.byName = %v, want strings without options", order, got)
		}
	}
}

// A type argument is written in the scope of its source and read inside the
// generic class, whose package may carry an enum of the same name. It travels as
// the enum it names: in a collection, an array and a bounded wildcard alike.
func TestResolve_AnEnumTypeArgumentKeepsTheEnumItNames(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"a/Envelope.java": `package a;

public class Envelope<T> { T data; }
`,
		"a/Status.java": `package a;

public enum Status { FROM_A }
`,
		"b/Status.java": `package b;

public enum Status { FROM_B }
`,
		"b/Holder.java": `package b;

import a.Envelope;
import java.util.List;

class Holder {
    Envelope<Status> one;
    Envelope<List<Status>> many;
    Envelope<Status[]> array;
    Envelope<? extends Status> wild;
}
`,
	})

	for _, field := range []string{"one", "many", "array", "wild"} {
		r := NewTypeResolver(classes)
		r.SetDependencyResolver(&libraryClasses{})
		data := r.Resolve("b.Holder", nil).Fields[field].Model.Fields["data"]
		if got := optionValues(data.Options); !reflect.DeepEqual(got, []any{"FROM_B"}) {
			t.Errorf("Holder.%s data options = %v, want FROM_B", field, got)
		}
	}
}

// A `@see` tag names an enum as a type does: in the scope of its source, so the
// Status nested in Order is not the one nested in Product, wherever the tag is
// written and however the name is spelled.
func TestSeeOptions_NestedEnumsOfOneNameAreTold(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"p/Declarations.java": `package p;

class Order {
    enum Status { OPEN, CLOSED }
    /** @see Status */
    String own;
}

class Product {
    enum Status { PHYSICAL, VIRTUAL }
    /** @see Status */
    String own;
}
`,
		"q/Holder.java": `package q;

import p.Order.Status;

class Holder {
    /** @see Status */
    String imported;
    /** @see p.Product.Status */
    String qualified;
    /** @see p.Order.Status */
    String order;
}
`,
	})

	for order, ordered := range map[string][]parser.Class{"as declared": classes, "reversed": reversed(classes)} {
		r := NewTypeResolver(ordered)
		for holder, wants := range map[string]map[string][]any{
			"p.Order":   {"own": {"OPEN", "CLOSED"}},
			"p.Product": {"own": {"PHYSICAL", "VIRTUAL"}},
			"q.Holder":  {"imported": {"OPEN", "CLOSED"}, "qualified": {"PHYSICAL", "VIRTUAL"}, "order": {"OPEN", "CLOSED"}},
		} {
			resolved := r.Resolve(holder, nil)
			for name, want := range wants {
				if got := optionValues(resolved.Fields[name].Options); !reflect.DeepEqual(got, want) {
					t.Errorf("%s: %s.%s options = %v, want %v", order, holder, name, got, want)
				}
			}
		}
	}
}

// An enum of a dependency and one of the project that carry one simple name are
// two declarations: the name goes to the one the scope of the source gives it,
// and that does not depend on which of them the resolver has read before.
func TestResolve_ADependencyEnumAndAProjectEnumOfOneNameAreTold(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"app/Status.java": `package app;

public enum Status { APP_ONLY }
`,
		"app/Local.java": `package app;

class Local {
    Status status;
}
`,
		"app/Imports.java": `package app;

import lib.Status;

class Imports {
    Status status;
}
`,
		"app/Qualified.java": `package app;

class Qualified {
    lib.Status status;
}
`,
	})
	newResolver := func() *TypeResolver {
		r := NewTypeResolver(classes)
		r.SetUnresolved(collector.NewUnresolvedSet())
		r.SetDependencyResolver(&libraryClasses{classes: map[string]parser.Class{
			"lib.Status": {Name: "Status", Package: "lib", IsEnum: true, EnumConstants: []string{"LIB_ONLY"}},
		}})
		return r
	}
	want := map[string]string{"app.Local": "APP_ONLY", "app.Imports": "LIB_ONLY", "app.Qualified": "LIB_ONLY"}

	orders := [][]string{{"app.Local", "app.Imports", "app.Qualified"}, {"app.Qualified", "app.Imports", "app.Local"}, {"app.Imports", "app.Local", "app.Qualified"}}
	for _, holders := range orders {
		r := newResolver()
		for _, holder := range holders {
			got := fieldFrom(r, holder, "status", false)
			if !wantConstants(got, want[holder]) {
				t.Errorf("reading %v: %s.status = %v, want a string of %s", holders, holder, got, want[holder])
			}
		}
	}
}

// An enum that declares @JsonValue is left unresolved, and an enum of the same
// constants beside it is not: not in another package, nor in another class of one
// file, nor in the dependencies.
func TestResolve_AnExcludedEnumDoesNotTakeAnEnumOfTheSameConstantsWithIt(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"a/Code.java": `package a;

import com.fasterxml.jackson.annotation.JsonValue;

public enum Code {
    X, Y;
    @JsonValue String text() { return null; }
}
`,
		"b/Code.java": `package b;

public enum Code { X, Y }
`,
		"a/Holder.java": `package a;

class Holder {
    Code code;
}
`,
		"b/Holder.java": `package b;

class Holder {
    Code code;
}
`,
		"c/Holder.java": `package c;

import lib.Code;

class Holder {
    Code code;
}
`,
		"d/Holder.java": `package d;

import a.Code;

class Holder {
    Code code;
    b.Code other;
}
`,
	})
	newResolver := func() *TypeResolver {
		r := NewTypeResolver(classes)
		r.SetUnresolved(collector.NewUnresolvedSet())
		r.SetDependencyResolver(&libraryClasses{classes: map[string]parser.Class{
			"lib.Code": {Name: "Code", Package: "lib", IsEnum: true, EnumConstants: []string{"X", "Y"}},
		}})
		return r
	}
	for _, binding := range []bool{false, true} {
		view := map[bool]string{false: "JSON", true: "binding"}[binding]
		for _, holders := range [][]string{{"a.Holder", "b.Holder", "c.Holder", "d.Holder"}, {"d.Holder", "c.Holder", "b.Holder", "a.Holder"}} {
			r := newResolver()
			for _, holder := range holders {
				for field, excluded := range map[string]bool{"code": holder == "a.Holder" || holder == "d.Holder", "other": false} {
					if holder != "d.Holder" && field == "other" {
						continue
					}
					got := fieldFrom(r, holder, field, binding)
					switch {
					case excluded && !binding && !wantUnresolved(got):
						t.Errorf("%s view, reading %v: %s.%s = %v, want it unresolved: a.Code declares @JsonValue", view, holders, holder, field, got)
					case (!excluded || binding) && !wantConstants(got, "X", "Y"):
						t.Errorf("%s view, reading %v: %s.%s = %v, want a string of X and Y", view, holders, holder, field, got)
					}
				}
			}
		}
	}
}

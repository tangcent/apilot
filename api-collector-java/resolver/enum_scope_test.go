package resolver

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// The tests below read what the export says of a field, as a short spelling:
//
//	str[A,B]  a string whose options are A and B
//	str       a string without options
//	int[0,1]  an integer whose options are 0 and 1
//	unres(X)  a type the resolver could not expand, named X
//	obj(X:a|b) an object of class X with the fields a and b
//	arr<..>, map<k,v>
//
// Every field is read twice, by a resolver of its own and by one that has read the
// fields before it, because what a name denotes must not depend on what the
// resolver has read.

type scopeRead struct {
	holder, field, want string
}

func describeType(m *model.ObjectModel) string {
	switch {
	case m == nil:
		return "<nil>"
	case m.Kind == model.KindArray:
		return "arr<" + describeType(m.Items) + ">"
	case m.Kind == model.KindMap:
		return "map<" + describeType(m.KeyModel) + "," + describeType(m.ValueModel) + ">"
	case m.Kind == model.KindObject:
		names := make([]string, 0, len(m.Fields))
		for name := range m.Fields {
			names = append(names, name)
		}
		sort.Strings(names)
		return "obj(" + m.TypeName + ":" + strings.Join(names, "|") + ")"
	case m.Kind == model.KindRef:
		return "ref(" + m.TypeName + ")"
	case m.TypeName == model.JsonTypeString:
		return "str"
	case m.TypeName == model.JsonTypeInt:
		return "int"
	}
	return "unres(" + m.TypeName + ")"
}

func describeField(f *model.FieldModel) string {
	if f == nil {
		return "<nofield>"
	}
	described := describeType(f.Model)
	if len(f.Options) > 0 {
		values := make([]string, len(f.Options))
		for i, option := range f.Options {
			values[i] = fmt.Sprint(option.Value)
		}
		if strings.HasPrefix(described, "arr<") {
			// The options belong to the items, as they are listed on the field.
			return described + "[" + strings.Join(values, ",") + "]"
		}
		described += "[" + strings.Join(values, ",") + "]"
	}
	return described
}

// libraryOf serves the declarations of the sources as the classes of a
// dependency: without a file path, and by qualified name only.
func libraryOf(t *testing.T, files map[string]string) *libraryClasses {
	t.Helper()
	library := &libraryClasses{classes: map[string]parser.Class{}}
	for _, c := range parseFiles(t, files) {
		c.FilePath = ""
		library.classes[c.QualifiedName()] = c
	}
	return library
}

// checkReads reads the fields in order, in the JSON view or in the binding view.
// library is nil for a run without a dependency resolver.
func checkReads(t *testing.T, label string, classes []parser.Class, library func() DependencyResolver, binding bool, reads ...scopeRead) {
	t.Helper()
	newResolver := func() *TypeResolver {
		r := NewTypeResolver(classes)
		r.SetUnresolved(collector.NewUnresolvedSet())
		if library != nil {
			r.SetDependencyResolver(library())
		}
		return r
	}
	read := func(r *TypeResolver, rd scopeRead) string {
		var resolved *model.ObjectModel
		if binding {
			resolved = r.ResolveForBinding(parser.Class{}, rd.holder, nil)
		} else {
			resolved = r.Resolve(rd.holder, nil)
		}
		if resolved == nil || resolved.Kind != model.KindObject {
			return "<holder: " + describeType(resolved) + ">"
		}
		return describeField(resolved.Fields[rd.field])
	}
	view := "JSON view"
	if binding {
		view = "binding view"
	}
	warm := newResolver()
	for _, rd := range reads {
		fresh := read(newResolver(), rd)
		if fresh != rd.want {
			t.Errorf("%s, %s: %s.%s = %s, want %s", label, view, rd.holder, rd.field, fresh, rd.want)
		}
		if got := read(warm, rd); got != fresh {
			t.Errorf("%s, %s: %s.%s = %s after reading the fields before it, and %s on a resolver of its own", label, view, rd.holder, rd.field, got, fresh)
		}
	}
}

func noLibrary() DependencyResolver { return &libraryClasses{} }

// A name only binds to an enum through the scope: member types, the file, a
// single-type import, the package, an import on demand. An import on demand of a
// package the sources do not hold, or java.lang, may carry the name as well, and
// the enum of some other package that shares it is not the type written. What a
// name nothing declares denotes is what it denotes without the enum: unresolved,
// or the scalar of the JDK type of that name (java.util.Currency, imported on
// demand, is a string, and has none of the constants of the enum of that name).
func TestResolve_ANameNoScopeDeclaresIsNotTakenForAnEnumOfThatName(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"other/Period.java":   "package other; public enum Period { DAY, WEEK }",
		"other/Currency.java": "package other; public enum Currency { CNY, USD }",
		"other/Date.java":     "package other; public enum Date { TODAY }",
		"other/Thread.java":   "package other; public enum Thread { MAIN }",
		"other/Status.java":   "package other; public enum Status { OTHER }",
		"app/Jdk.java": `package app;
import java.time.*;
import java.util.*;
class Jdk { Period period; Currency currency; Date date; Thread thread; LocalDate day; }
`,
		"app/Single.java": `package app;
import java.time.Period;
class Single { Period period; }
`,
		"app/Unscanned.java": `package app;
import thirdparty.model.*;
class Unscanned { Status status; java.util.List<Status> statuses; Status[] array; }
`,
	})
	for _, library := range []func() DependencyResolver{nil, noLibrary} {
		for _, binding := range []bool{false, true} {
			checkReads(t, "an import on demand", classes, library, binding,
				scopeRead{"app.Jdk", "currency", "str"},
				scopeRead{"app.Jdk", "thread", "unres(Thread)"},
				scopeRead{"app.Unscanned", "status", "unres(Status)"},
				scopeRead{"app.Unscanned", "array", "arr<unres(Status)>"},
			)
		}
		// The JDK value types keep their scalar, in the JSON view.
		checkReads(t, "an import on demand", classes, library, false,
			scopeRead{"app.Jdk", "period", "str"},
			scopeRead{"app.Jdk", "date", "str"},
			scopeRead{"app.Jdk", "day", "str"},
			scopeRead{"app.Single", "period", "str"},
		)
	}
}

// An import of a package of the dependencies, on demand or of the class by name,
// names the enum of the library, not a project enum that carries the simple name.
// Which of them the resolver has read does not change that.
func TestResolve_AnEnumOfTheDependenciesIsNotReplacedByAProjectEnumOfItsName(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"other/Status.java": "package other; public enum Status { OTHER }",
		"app/Star.java":     "package app; import lib.*; class Star { Status s; }",
		"app/Single.java":   "package app; import lib.Status; class Single { Status s; }",
	})
	library := func() DependencyResolver {
		return libraryOf(t, map[string]string{"lib/Status.java": "package lib; public enum Status { LIB }"})
	}
	checkReads(t, "an import of lib", classes, library, false,
		scopeRead{"app.Star", "s", "str[LIB]"},
		scopeRead{"app.Single", "s", "str[LIB]"},
		scopeRead{"app.Star", "s", "str[LIB]"},
	)
}

// A class read from a library is a compilation unit of its own: the Status its
// field names is the Status of its package, not the Status of the project, and the
// enum nested in it is not the enum of the project that carries its name. Whether
// the library's enum has been read does not change that.
func TestResolve_AClassOfALibraryDoesNotTakeTheEnumsOfTheProject(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"app/Status.java": "package app; public enum Status { APP }",
		"app/Kind.java":   "package app; public enum Kind { KAPP }",
		"app/H.java":      "package app; import lib.Order; import lib.Ticket; class H { Order o; Ticket t; }",
	})
	library := func() DependencyResolver {
		return libraryOf(t, map[string]string{
			"lib/Order.java":  "package lib; import java.util.List; public class Order { public Status status; public List<Status> list; }",
			"lib/Status.java": "package lib; public enum Status { LIB }",
			"lib/Ticket.java": "package lib; public class Ticket { public enum Kind { K1 } public Kind kind; }",
		})
	}
	for _, warm := range []bool{false, true} {
		r := NewTypeResolver(classes)
		r.SetUnresolved(collector.NewUnresolvedSet())
		r.SetDependencyResolver(library())
		if warm {
			// The library's enum has been read already.
			r.Resolve("lib.Status", nil)
		}
		h := r.Resolve("app.H", nil)
		if got := describeField(h.Fields["o"].Model.Fields["status"]); got != "str[LIB]" {
			t.Errorf("warm=%v: lib.Order.status = %s, want str[LIB], not the enum of the project", warm, got)
		}
		if got := describeField(h.Fields["o"].Model.Fields["list"]); got != "arr<str>[LIB]" {
			t.Errorf("warm=%v: lib.Order.list = %s, want arr<str>[LIB]", warm, got)
		}
		if got := describeField(h.Fields["t"].Model.Fields["kind"]); got != "unres(Kind)" {
			t.Errorf("warm=%v: lib.Ticket.kind = %s, want unres(Kind), not the app.Kind of the project", warm, got)
		}
	}
}

// A member type the class inherits, from a superclass or an interface, however far
// up, shadows what the package declares, as one it declares itself does (JLS 6.4.1,
// 8.5): inside Sub, Status is the Status nested in Outer, not the one of the
// package. The classes around a class lend their member types too, with the ones
// they inherit. The supertypes of a class are named in the scope around its header,
// where its own member types are not in scope.
func TestResolve_AMemberTypeThatIsInheritedShadowsTheTypesOfThePackage(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"p/Status.java":   "package p; public enum Status { TOP }",
		"p/Outer.java":    "package p; public class Outer { public enum Status { OUTER } }",
		"p/Sub.java":      "package p; class Sub extends Outer { Status s; }",
		"q/Sub2.java":     "package q; import p.Outer; class Sub2 extends Outer { Status s; }",
		"p/GrandSub.java": `package p; class GrandSub extends Sub { Status s; }`,
		"p/Ifaces.java": `package p;
interface HasStatus { enum Status { IFACE } }
interface Deeper extends HasStatus { }
class ViaIface implements HasStatus { Status s; }
class ViaDeeper implements Deeper { Status s; }
`,
		"p/Around.java": `package p;
class Wrapper extends Outer { class Inner { Status s; } }
class Own extends Outer { enum Status { OWN } Status s; }
`,
		"p/Header.java": `package p;
class Parent { enum Status { PARENT } }
class Header extends Parent { enum Parent { X } Status s; }
`,
	})
	reads := []scopeRead{
		{"p.Sub", "s", "str[OUTER]"},
		{"q.Sub2", "s", "str[OUTER]"},
		{"p.GrandSub", "s", "str[OUTER]"},
		{"p.ViaIface", "s", "str[IFACE]"},
		{"p.ViaDeeper", "s", "str[IFACE]"},
		{"p.Wrapper.Inner", "s", "str[OUTER]"},
		{"p.Own", "s", "str[OWN]"},
		{"p.Header", "s", "str[PARENT]"},
	}
	for order, ordered := range map[string][]parser.Class{"as declared": classes, "reversed": reversed(classes)} {
		for _, library := range []func() DependencyResolver{nil, noLibrary} {
			checkReads(t, order, ordered, library, false, reads...)
			checkReads(t, order, ordered, library, true, reads...)
		}
	}
}

// A member type that is a class and not an enum shadows the types of the package
// too: inside Outer, the nested Code is the class Outer declares, never the
// top-level enum or class of the package that carries the name, nor the class
// nested in another class that carries it as well.
func TestResolve_AMemberClassIsNeverReplacedByAnotherDeclarationOfItsName(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"d/Code.java":  "package d; public enum Code { E1 }",
		"d/Outer.java": "package d; public class Outer { public static class Code { String v; } Code inner; }",
		"d/Item.java":  "package d; public class Item { String top; }",
		"d/Holder.java": `package d;
class Holder { Item item; Code code; }
class Box { static class Item { String nested; } Item item; }
`,
		"d/A.java": "package d; class A { static class Entry { String a; } Entry entry; }",
		"d/B.java": "package d; class B { static class Entry { String b; } Entry entry; }",
	})
	for order, ordered := range map[string][]parser.Class{"as declared": classes, "reversed": reversed(classes)} {
		for _, library := range []func() DependencyResolver{nil, noLibrary} {
			checkReads(t, order, ordered, library, false,
				scopeRead{"d.Holder", "item", "obj(Item:top)"},
				scopeRead{"d.Holder", "code", "str[E1]"},
				scopeRead{"d.Outer", "inner", "obj(Code:v)"},
				scopeRead{"d.Box", "item", "obj(Item:nested)"},
				// Two classes nested in different classes are two classes.
				scopeRead{"d.A", "entry", "obj(Entry:a)"},
				scopeRead{"d.B", "entry", "obj(Entry:b)"},
			)
		}
	}
}

// An enum of the package of the source is the enum the name denotes, whether the
// current view reads it or not, and a class of another package that carries the
// name and that nothing imports is not taken for it: not for the name written
// directly, in a collection, or as a type argument (which is qualified where it is
// written).
func TestResolve_AnEnumOfThePackageIsNotReplacedByAnUnrelatedClassOfItsName(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"com/ex/unrelated/Status.java": "package com.ex.unrelated; public class Status { private String unrelatedLabel; }",
		"com/ex/gen/Box.java":          "package com.ex.gen; public class Box<T> { public T data; }",
		"com/ex/app/dto/Status.java":   "package com.ex.app.dto; public enum Status { OPEN, CLOSED }",
		"com/ex/app/dto/Ctl.java": `package com.ex.app.dto;
import com.ex.gen.Box;
import java.util.List;
class Ctl { Status direct; List<Status> list; Box<Status> boxed; }
`,
		"com/ex/app/dto/Coded.java": `package com.ex.app.dto;
import com.fasterxml.jackson.annotation.JsonValue;
public enum Coded { X; @JsonValue String v() { return null; } }
class UsesCoded { Coded coded; }
`,
	})
	for order, ordered := range map[string][]parser.Class{"as declared": classes, "reversed": reversed(classes)} {
		for _, binding := range []bool{false, true} {
			checkReads(t, order, ordered, noLibrary, binding,
				scopeRead{"com.ex.app.dto.Ctl", "direct", "str[OPEN,CLOSED]"},
			)
		}
		checkReads(t, order, ordered, nil, false,
			scopeRead{"com.ex.app.dto.Ctl", "list", "arr<str>[OPEN,CLOSED]"},
			// An enum that the JSON view leaves out is no less the enum the name denotes.
			scopeRead{"com.ex.app.dto.UsesCoded", "coded", "unres(Coded)"},
		)
		r := NewTypeResolver(ordered)
		r.SetDependencyResolver(&libraryClasses{})
		boxed := r.Resolve("com.ex.app.dto.Ctl", nil).Fields["boxed"]
		if got := describeField(boxed.Model.Fields["data"]); got != "str[OPEN,CLOSED]" {
			t.Errorf("%s: Ctl.boxed.data = %s, want str[OPEN,CLOSED], not the class of com.ex.unrelated", order, got)
		}
	}
}

// The form Jackson writes an enum in, as the source tells it. A field typed with an
// enum written as the name of a constant is a string of the names; one written as
// anything else is the type name, as a type that was not read: a member of the enum
// or of an interface it implements that carries @JsonValue, a name @JsonProperty
// gives a constant, the shape OBJECT, ARRAY or a number, and a serializer. A
// @JsonFormat or a @JsonSerialize on the field decides for the enums it holds, a
// collection of them included. Spring binds by the name of the constant whatever
// the form, so the binding view reads the names as they are in the source.
func TestResolve_AnEnumOfACustomJacksonFormKeepsItsTypeName(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"p/PlainBodies.java": "package p; public enum PlainBodies { A { void f() {} }, B { void f() {} }; abstract void f(); }",
		"p/WithCtor.java":    "package p; public enum WithCtor { A(\"a\"), B(\"b\"); private final String c; WithCtor(String c) { this.c = c; } public String getC() { return c; } }",
		"p/Creator.java":     "package p; import com.fasterxml.jackson.annotation.JsonCreator; public enum Creator { A, B; @JsonCreator public static Creator of(String s) { return A; } }",
		"p/ValM.java":        "package p; import com.fasterxml.jackson.annotation.JsonValue; public enum ValM { A, B; @JsonValue public String v() { return \"x\"; } }",
		"p/ValOff.java":      "package p; import com.fasterxml.jackson.annotation.JsonValue; public enum ValOff { A; @JsonValue(false) public String v() { return \"x\"; } }",
		"p/Coded.java":       "package p; import com.fasterxml.jackson.annotation.JsonValue; public interface Coded { @JsonValue String code(); }",
		"p/ViaIface.java":    "package p; public enum ViaIface implements Coded { A; public String code() { return \"x\"; } }",
		"p/Chain.java": `package p;
import java.io.Serializable;
interface Sub extends Coded { }
public enum ViaSuper implements Serializable, Sub { A; public String code() { return "x"; } }
interface Plain { String label(); }
public enum ViaPlain implements Plain { A; public String label() { return "x"; } }
`,
		"p/Shapes.java": `package p;
import com.fasterxml.jackson.annotation.JsonFormat;
@JsonFormat(shape = JsonFormat.Shape.OBJECT) public enum ShapeObj { A }
@JsonFormat(shape = JsonFormat.Shape.NUMBER_INT) public enum ShapeNum { A, B }
@JsonFormat(shape = JsonFormat.Shape.ARRAY) public enum ShapeArr { A }
@JsonFormat(shape = JsonFormat.Shape.STRING) public enum ShapeStr { A }
`,
		"p/PropConst.java": `package p;
import com.fasterxml.jackson.annotation.JsonProperty;
public enum PropConst { A, @JsonProperty("b_x") B, @JsonProperty("") C, @JsonProperty(value = "d_x") D }
public enum EmptyName { A, @JsonProperty("") B }
`,
		"p/SerUsing.java": "package p; import com.fasterxml.jackson.databind.annotation.JsonSerialize; @JsonSerialize(using = Foo.class) public enum SerUsing { A }",
		"p/Alias.java":    "package p; import com.fasterxml.jackson.annotation.JsonAlias; public enum Alias { @JsonAlias(\"x\") A }",
		"p/Holder.java": `package p;
import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;
import java.util.List;
import java.util.Map;
class Holder {
  PlainBodies plainBodies; WithCtor withCtor; Creator creator; ValM valM; ValOff valOff;
  ViaIface viaIface; ViaSuper viaSuper; ViaPlain viaPlain;
  ShapeObj shapeObj; ShapeNum shapeNum; ShapeArr shapeArr; ShapeStr shapeStr;
  PropConst propConst; List<PropConst> propConsts; EmptyName emptyName; SerUsing serUsing; Alias alias;
  @JsonFormat(shape = JsonFormat.Shape.NUMBER) PlainBodies fieldNumber;
  @JsonFormat(shape = JsonFormat.Shape.NUMBER_INT) List<WithCtor> listNumber;
  @JsonFormat(shape = JsonFormat.Shape.NUMBER) WithCtor[] arrayNumber;
  @JsonFormat(shape = JsonFormat.Shape.ARRAY) PlainBodies fieldArray;
  @JsonFormat(shape = JsonFormat.Shape.STRING) ShapeNum fieldString;
  @JsonFormat(shape = JsonFormat.Shape.STRING) PlainBodies fieldStringOfPlain;
  @JsonFormat(shape = JsonFormat.Shape.NUMBER) ValM fieldNumberOfValue;
  @JsonSerialize(using = Foo.class) PlainBodies fieldSerialized;
  @JsonSerialize(contentUsing = Foo.class) List<PlainBodies> contentSerialized;
  @JsonFormat(shape = JsonFormat.Shape.NUMBER) Map<String, WithCtor> mapNumber;
  @JsonFormat(shape = JsonFormat.Shape.NUMBER) Box boxed;
}
class Box { PlainBodies inside; }
`,
	})
	json := []scopeRead{
		{"p.Holder", "plainBodies", "str[A,B]"},
		{"p.Holder", "withCtor", "str[A,B]"},
		{"p.Holder", "creator", "str[A,B]"},
		{"p.Holder", "valM", "unres(ValM)"},
		{"p.Holder", "valOff", "str[A]"},
		{"p.Holder", "viaIface", "unres(ViaIface)"},
		{"p.Holder", "viaSuper", "unres(ViaSuper)"},
		{"p.Holder", "viaPlain", "str[A]"},
		{"p.Holder", "shapeObj", "unres(ShapeObj)"},
		{"p.Holder", "shapeNum", "unres(ShapeNum)"},
		{"p.Holder", "shapeArr", "unres(ShapeArr)"},
		{"p.Holder", "shapeStr", "str[A]"},
		{"p.Holder", "propConst", "unres(PropConst)"},
		{"p.Holder", "propConsts", "arr<unres(PropConst)>"},
		{"p.Holder", "emptyName", "str[A,B]"},
		{"p.Holder", "serUsing", "unres(SerUsing)"},
		{"p.Holder", "alias", "str[A]"},
		{"p.Holder", "fieldNumber", "unres(PlainBodies)"},
		{"p.Holder", "listNumber", "arr<unres(WithCtor)>"},
		{"p.Holder", "arrayNumber", "arr<unres(WithCtor)>"},
		{"p.Holder", "fieldArray", "unres(PlainBodies)"},
		// The string shape of a field does not turn back an enum that is written as
		// something else.
		{"p.Holder", "fieldString", "unres(ShapeNum)"},
		{"p.Holder", "fieldStringOfPlain", "str[A,B]"},
		{"p.Holder", "fieldNumberOfValue", "unres(ValM)"},
		{"p.Holder", "fieldSerialized", "unres(PlainBodies)"},
		{"p.Holder", "contentSerialized", "arr<unres(PlainBodies)>"},
		{"p.Holder", "mapNumber", "map<str,unres(WithCtor)>"},
		// The shape of a field is the shape of what it holds, not of the fields of an
		// object it holds.
		{"p.Box", "inside", "str[A,B]"},
	}
	binding := []scopeRead{
		{"p.Holder", "valM", "str[A,B]"},
		{"p.Holder", "viaIface", "str[A]"},
		{"p.Holder", "shapeObj", "str[A]"},
		{"p.Holder", "shapeNum", "str[A,B]"},
		{"p.Holder", "propConst", "str[A,B,C,D]"},
		{"p.Holder", "fieldNumber", "str[A,B]"},
	}
	for order, ordered := range map[string][]parser.Class{"as declared": classes, "reversed": reversed(classes)} {
		for _, library := range []func() DependencyResolver{nil, noLibrary} {
			checkReads(t, order, ordered, library, false, json...)
			checkReads(t, order, ordered, library, true, binding...)
		}
	}
}

// An interface of a library that an enum implements carries @JsonValue on a method
// as well: the dependencies are asked for it, and without them the enum keeps the
// form its own source tells.
func TestResolve_AnEnumImplementingAnInterfaceOfALibraryFollowsItsJsonValue(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"app/Kind.java":   "package app; import lib.Coded; public enum Kind implements Coded { A; public String code() { return null; } }",
		"app/Holder.java": "package app; class Holder { Kind kind; }",
	})
	library := func() DependencyResolver {
		return libraryOf(t, map[string]string{
			"lib/Coded.java": "package lib; import com.fasterxml.jackson.annotation.JsonValue; public interface Coded { @JsonValue String code(); }",
		})
	}
	checkReads(t, "a library interface", classes, library, false, scopeRead{"app.Holder", "kind", "unres(Kind)"})
	checkReads(t, "no dependencies", classes, nil, false, scopeRead{"app.Holder", "kind", "str[A]"})
}

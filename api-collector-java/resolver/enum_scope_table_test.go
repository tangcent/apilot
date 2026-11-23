package resolver

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// Tables of the scopes an enum name is written in, each read in the JSON view with
// and without a dependency resolver, on a resolver of its own and on one that has
// read the fields before it (see checkReads). The sources stand for what a project
// holds; a name that needs an import of a package the sources do not hold, a
// static import, or a supertype the sources do not parse is left unresolved, never
// bound to another enum.

func checkTable(t *testing.T, label string, files map[string]string, reads ...scopeRead) {
	t.Helper()
	classes := parseFiles(t, files)
	for order, ordered := range map[string][]parser.Class{"as declared": classes, "reversed": reversed(classes)} {
		for _, library := range []func() DependencyResolver{nil, noLibrary} {
			checkReads(t, label+", "+order, ordered, library, false, reads...)
		}
	}
}

// Nested enums of one name in classes of one file: through the class, a collection,
// an array, a map, a set, a name written in full, and from inside a class nested in
// the class that nests the enum.
func TestResolve_NestedEnumsOfOneNameInOneFileAreToldInEveryPosition(t *testing.T) {
	checkTable(t, "nested enums of one file", map[string]string{
		"p/Models.java": `package p;
class Order { enum Status { A, B }
  Status own;
  class Inner { Status viaOuter; }
  static class SInner { Status viaOuterStatic; }
}
class Product { enum Status { C } Status own; }
`,
		"p/Holder.java": `package p;
import java.util.*;
class Holder {
  Order.Status a; Product.Status b;
  List<Order.Status> la; Order.Status[] aa; Map<String, Product.Status> mb; Map<Product.Status, String> kb;
  Set<Product.Status> sp;
  p.Order.Status qa;
}
`,
	},
		scopeRead{"p.Holder", "a", "str[A,B]"}, scopeRead{"p.Holder", "b", "str[C]"},
		scopeRead{"p.Holder", "la", "arr<str>[A,B]"}, scopeRead{"p.Holder", "aa", "arr<str>[A,B]"},
		scopeRead{"p.Holder", "mb", "map<str,str>"}, scopeRead{"p.Holder", "kb", "map<str,str>"},
		scopeRead{"p.Holder", "sp", "arr<str>[C]"}, scopeRead{"p.Holder", "qa", "str[A,B]"},
		scopeRead{"p.Order", "own", "str[A,B]"}, scopeRead{"p.Product", "own", "str[C]"},
		scopeRead{"p.Order.Inner", "viaOuter", "str[A,B]"}, scopeRead{"p.Order.SInner", "viaOuterStatic", "str[A,B]"},
	)
}

// An enum nested in the Order of one package or another, reached through an import
// of the class, of the package, of the enum, of the members of the class, or named
// in full. A static import is not read: the name stays unresolved.
func TestResolve_NestedEnumsOfOneNameInPackagesAreToldByTheirImports(t *testing.T) {
	checkTable(t, "nested enums of two packages", map[string]string{
		"a/Order.java":   "package a; public class Order { public enum Status { A } }",
		"b/Order.java":   "package b; public class Order { public enum Status { B } }",
		"b/HolderB.java": "package b; class HolderB { Order.Status f; }",
		"a/HolderA.java": "package a; class HolderA { Order.Status f; }",
		"c/Imp1.java":    "package c; import a.Order; class Imp1 { Order.Status f; }",
		"c/Imp2.java":    "package c; import b.*; class Imp2 { Order.Status f; }",
		"c/Imp3.java":    "package c; import a.Order.Status; class Imp3 { Status f; }",
		"c/Imp4.java":    "package c; import b.Order.*; class Imp4 { Status f; }",
		"c/Imp5.java":    "package c; class Imp5 { a.Order.Status f; b.Order.Status g; }",
		"c/Imp6.java":    "package c; import static a.Order.Status; class Imp6 { Status f; }",
	},
		scopeRead{"b.HolderB", "f", "str[B]"}, scopeRead{"a.HolderA", "f", "str[A]"},
		scopeRead{"c.Imp1", "f", "str[A]"}, scopeRead{"c.Imp2", "f", "str[B]"}, scopeRead{"c.Imp3", "f", "str[A]"},
		scopeRead{"c.Imp4", "f", "str[B]"}, scopeRead{"c.Imp5", "f", "str[A]"}, scopeRead{"c.Imp5", "g", "str[B]"},
		scopeRead{"c.Imp6", "f", "unres(Status)"},
	)
}

// An enum and a class of one name in different packages: the scope gives the
// declaration, whichever kind it is.
func TestResolve_AnEnumAndAClassOfOneNameInPackagesAreToldByTheirScope(t *testing.T) {
	checkTable(t, "an enum and a class of one name", map[string]string{
		"a/Code.java": "package a; public enum Code { A }",
		"b/Code.java": "package b; public class Code { String v; }",
		"a/H.java":    "package a; class H { Code f; }",
		"b/H.java":    "package b; class H { Code f; }",
		"c/H1.java":   "package c; import b.Code; class H1 { Code f; }",
		"c/H2.java":   "package c; import b.*; class H2 { Code f; }",
		"c/H3.java":   "package c; import a.*; class H3 { Code f; }",
		"c/H4.java":   "package c; import a.Code; class H4 { Code f; }",
	},
		scopeRead{"a.H", "f", "str[A]"}, scopeRead{"b.H", "f", "obj(Code:v)"}, scopeRead{"c.H1", "f", "obj(Code:v)"},
		scopeRead{"c.H2", "f", "obj(Code:v)"}, scopeRead{"c.H3", "f", "str[A]"}, scopeRead{"c.H4", "f", "str[A]"},
	)
}

// An enum at the end of every container the export knows, in one position or two.
// The values of a map are strings of the constants and the map lists none; Optional,
// Iterable and EnumSet are not unwrapped for any type.
func TestResolve_AnEnumInEveryContainerListsItsConstants(t *testing.T) {
	checkTable(t, "containers", map[string]string{
		"p/Status.java": "package p; public enum Status { A, B }",
		"p/Holder.java": `package p;
import java.util.*;
class Holder {
  Status f; List<Status> l; Set<Status> s; Status[] a; Status[][] aa; Collection<Status> c; ArrayList<Status> al;
  Map<String, Status> mv; Map<Status, String> mk; Map<Status, List<Status>> mkv;
  List<? extends Status> up; List<? super Status> lo; List<?> any; List<List<Status>> ll; Map<String, List<Status>> mlist;
  Collection<Set<Status>> cs; List<Status>[] la;
}
`,
	},
		scopeRead{"p.Holder", "f", "str[A,B]"}, scopeRead{"p.Holder", "l", "arr<str>[A,B]"}, scopeRead{"p.Holder", "s", "arr<str>[A,B]"},
		scopeRead{"p.Holder", "a", "arr<str>[A,B]"}, scopeRead{"p.Holder", "aa", "arr<arr<str>>[A,B]"}, scopeRead{"p.Holder", "c", "arr<str>[A,B]"},
		scopeRead{"p.Holder", "al", "arr<str>[A,B]"}, scopeRead{"p.Holder", "mv", "map<str,str>"}, scopeRead{"p.Holder", "mk", "map<str,str>"},
		scopeRead{"p.Holder", "mkv", "map<str,arr<str>>"},
		scopeRead{"p.Holder", "up", "arr<str>[A,B]"}, scopeRead{"p.Holder", "lo", "arr<obj(object:)>"}, scopeRead{"p.Holder", "any", "arr<obj(object:)>"},
		scopeRead{"p.Holder", "ll", "arr<arr<str>>[A,B]"}, scopeRead{"p.Holder", "mlist", "map<str,arr<str>>"},
		scopeRead{"p.Holder", "cs", "arr<arr<str>>[A,B]"}, scopeRead{"p.Holder", "la", "arr<arr<str>>[A,B]"},
	)
}

// Declarations that look alike without being one: an enum declared by two files of
// one package is the one the registry holds, and enums nested two deep, in a record,
// in an interface, in a generic class, in the default package, or beside a
// top-level enum of the name are each told by their class.
func TestResolve_EnumsNestedDeepOrBesideOneOfTheNameAreToldByTheirClass(t *testing.T) {
	checkTable(t, "duplicates and nesting", map[string]string{
		"src/p/Status.java": "package p; public enum Status { A, B }",
		"gen/p/Status.java": "package p; public enum Status { A, B }",
		"p/UseStatus.java":  "package p; class UseStatus { Status s; }",
		"p/N.java": `package p;
class Outer { class Mid { enum Deep { D1 } } Mid.Deep viaMid; }
class Other { class Mid { enum Deep { D2 } } }
interface Consts { enum Kind { IK } }
record Rec(Kind k) { enum Kind { RK } }
class Box<T> { enum Kind { BK } Kind k; T t; }
enum Top { T1 }
class SameFile { enum Top { NT } Top own; }
class SameFile2 { Top outerTop; }
`,
		"p/UseN.java": "package p; class UseN { Outer.Mid.Deep d1; Other.Mid.Deep d2; Consts.Kind ik; Rec.Kind rk; Box.Kind bk; Top top; SameFile.Top nt; }",
	},
		scopeRead{"p.UseStatus", "s", "str[A,B]"},
		scopeRead{"p.UseN", "d1", "str[D1]"}, scopeRead{"p.UseN", "d2", "str[D2]"}, scopeRead{"p.UseN", "ik", "str[IK]"},
		scopeRead{"p.UseN", "rk", "str[RK]"}, scopeRead{"p.UseN", "bk", "str[BK]"}, scopeRead{"p.UseN", "top", "str[T1]"},
		scopeRead{"p.UseN", "nt", "str[NT]"}, scopeRead{"p.Box", "k", "str[BK]"}, scopeRead{"p.SameFile", "own", "str[NT]"},
		scopeRead{"p.SameFile2", "outerTop", "str[T1]"}, scopeRead{"p.Outer", "viaMid", "str[D1]"},
	)
	checkTable(t, "the default package", map[string]string{
		"Status.java": "public enum Status { TOP }",
		"Outer.java":  "public class Outer { public enum Status { NESTED } Status own; }",
		"Use.java":    "class Use { Status top; Outer.Status nested; }",
	},
		scopeRead{"Outer", "own", "str[NESTED]"}, scopeRead{"Use", "top", "str[TOP]"}, scopeRead{"Use", "nested", "str[NESTED]"},
	)
	checkTable(t, "a type variable named like an enum", map[string]string{
		"p/Kind.java": "package p; public enum Kind { K }",
		"p/Box.java":  "package p; public class Box<Kind> { public Kind value; }",
		"p/H.java":    "package p; class H { Box<String> b; }",
	},
		scopeRead{"p.H", "b", "obj(Box:value)"},
	)
}

// Names of one simple name in the classes of one file and of the package, in every
// class that writes it, as a getter and as a record component.
func TestResolve_EnumsOfOneNameAreToldInGettersAndRecordComponents(t *testing.T) {
	checkTable(t, "getters and records", map[string]string{
		"p/Order.java":   "package p; public class Order { public enum Status { OA, OB } }",
		"p/Product.java": "package p; public class Product { public enum Status { PC } }",
		"p/Status.java":  "package p; public enum Status { TOP }",
		"p/H.java": `package p; import java.util.*;
class H {
  Order.Status o; Product.Status pr; Status top;
  public Status getDerived() { return null; }
  public List<Order.Status> getDerivedList() { return null; }
  public Product.Status getDerivedNested() { return null; }
}`,
		"p/R.java": "package p; import java.util.*; record R(Status s, List<Order.Status> l, Product.Status n) {}",
	},
		scopeRead{"p.H", "o", "str[OA,OB]"}, scopeRead{"p.H", "pr", "str[PC]"}, scopeRead{"p.H", "top", "str[TOP]"},
		scopeRead{"p.H", "derived", "str[TOP]"}, scopeRead{"p.H", "derivedList", "arr<str>[OA,OB]"}, scopeRead{"p.H", "derivedNested", "str[PC]"},
		scopeRead{"p.R", "s", "str[TOP]"}, scopeRead{"p.R", "l", "arr<str>[OA,OB]"}, scopeRead{"p.R", "n", "str[PC]"},
	)
}

// A `@see` names an enum through the scope of its class: imported, written in full,
// nested in a class, or a member of an imported class; a class of the same name in
// the package, which is no enum, gives no options, and neither does a name that no
// scope binds. A class inherits the member types of a supertype written in full.
func TestSeeOptions_NamesFollowTheScopeOfTheirClass(t *testing.T) {
	checkTable(t, "@see", map[string]string{
		"a/Status.java": "package a; public enum Status { AS }",
		"b/Status.java": "package b; public enum Status { BS }",
		"p/Status.java": "package p; public class Status { String v; }",
		"p/Outer.java":  "package p; public class Outer { public enum Status { OS } }",
		"p/Other.java":  "package p; public class Other { public enum Status { OT } }",
		"p/Holder.java": `package p;
import a.Status;
class Holder {
  /** @see Status */ String imported;
  /** @see a.Status */ String qualified;
  /** @see b.Status */ String qualifiedB;
  /** @see Outer.Status */ String nested;
  /** @see p.Outer.Status */ String nestedQualified;
  /** @see Other.Status */ String nestedOther;
  /** @see Status#AS */ String member;
}
`,
		"p/Holder2.java": `package p;
class Holder2 {
  /** @see Status */ String unrelatedClass;
  /** @see Outer.Status */ String nested;
  /** @see b.Status */ String qualifiedB;
}
`,
		"p/Holder3.java": `package p;
import b.*;
class Holder3 {
  /** @see Status */ String viaDemand;
}
`,
		"q/Holder4.java": `package q;
import b.*;
import p.Outer;
class Holder4 {
  /** @see Status */ String demand;
  /** @see Outer.Status */ String nested;
}
`,
		"q/Holder5.java": `package q;
class Holder5 extends p.Outer {
  /** @see Status */ String inherited;
}
`,
	},
		scopeRead{"p.Holder", "imported", "str[AS]"}, scopeRead{"p.Holder", "qualified", "str[AS]"}, scopeRead{"p.Holder", "qualifiedB", "str[BS]"},
		scopeRead{"p.Holder", "nested", "str[OS]"}, scopeRead{"p.Holder", "nestedQualified", "str[OS]"}, scopeRead{"p.Holder", "nestedOther", "str[OT]"},
		scopeRead{"p.Holder", "member", "str"},
		scopeRead{"p.Holder2", "unrelatedClass", "str"},
		scopeRead{"p.Holder2", "nested", "str[OS]"}, scopeRead{"p.Holder2", "qualifiedB", "str[BS]"},
		scopeRead{"p.Holder3", "viaDemand", "str"},
		scopeRead{"q.Holder4", "demand", "str[BS]"}, scopeRead{"q.Holder4", "nested", "str[OS]"},
		scopeRead{"q.Holder5", "inherited", "str[OS]"},
	)
}

// An enum of the project and one of the dependencies that carry one name, with
// @JsonValue on either: the scope gives the one, and the form it is written in is
// its own.
func TestResolve_EnumsOfTheDependenciesAndOfTheProjectAreToldWhateverTheirForm(t *testing.T) {
	classes := parseFiles(t, map[string]string{
		"app/Status.java":       "package app; public enum Status { APP }",
		"app/Codey.java":        "package app; import com.fasterxml.jackson.annotation.JsonValue; public enum Codey { AC; @JsonValue String v() { return \"x\"; } }",
		"app/ImportsLib.java":   "package app; import lib.Status; class ImportsLib { Status s; lib.Status q; java.util.List<lib.Status> l; }",
		"app/Local.java":        "package app; class Local { Status s; }",
		"app/ImportsCodey.java": "package app; import lib.Codey; class ImportsCodey { Codey c; }",
		"app/LocalCodey.java":   "package app; class LocalCodey { Codey c; }",
	})
	library := func() DependencyResolver {
		return libraryOf(t, map[string]string{
			"lib/Status.java": "package lib; import com.fasterxml.jackson.annotation.JsonValue; public enum Status { LIB; @JsonValue String v() { return \"x\"; } }",
			"lib/Codey.java":  "package lib; public enum Codey { LC }",
		})
	}
	checkReads(t, "enums of the dependencies", classes, library, false,
		scopeRead{"app.ImportsLib", "s", "unres(Status)"},
		scopeRead{"app.ImportsLib", "q", "unres(lib.Status)"},
		scopeRead{"app.Local", "s", "str[APP]"},
		scopeRead{"app.ImportsCodey", "c", "str[LC]"},
		scopeRead{"app.LocalCodey", "c", "unres(Codey)"},
	)
}

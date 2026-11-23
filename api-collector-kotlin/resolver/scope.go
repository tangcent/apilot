package resolver

import (
	"strings"

	javaparser "github.com/tangcent/apilot/api-collector-java/parser"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
)

// builtinType is a type of the Kotlin standard library or the JDK that the shared
// resolver reads under a name of its own, or a class that a renaming import
// names, which it reads by its own name.
type builtinType struct {
	// name is the class the resolver reads it as, written in full: a compilation
	// unit of the Java resolver cannot bind a qualified name to another class, as
	// it could a simple one.
	name string
	// element is the argument an array type of a primitive carries when it is
	// written without one: IntArray is a list of Integer.
	element string
}

// builtinTypes holds the types the adapter maps, by the qualified name Kotlin
// gives them, to the class of the JDK that the shared resolver knows.
var builtinTypes = func() map[string]builtinType {
	types := map[string]builtinType{}
	add := func(pkg, name, element string, names ...string) {
		for _, simple := range names {
			types[pkg+"."+simple] = builtinType{name: name, element: element}
		}
	}

	// kotlin.*
	add("kotlin", "java.lang.Integer", "", "Int", "Byte", "Short", "UByte", "UShort", "UInt")
	add("kotlin", "java.lang.Long", "", "Long", "ULong")
	add("kotlin", "java.lang.Float", "", "Float")
	add("kotlin", "java.lang.Double", "", "Double")
	add("kotlin", "java.lang.Boolean", "", "Boolean")
	add("kotlin", "java.lang.String", "", "Char", "String")
	add("kotlin", "java.lang.Number", "", "Number")
	add("kotlin", "java.lang.Object", "", "Any")
	add("kotlin", "java.lang.Void", "", "Unit", "Nothing")
	add("kotlin", "java.util.List", "", "Array")
	add("kotlin", "java.util.List", "java.lang.Integer", "IntArray", "ByteArray", "ShortArray", "UByteArray", "UShortArray", "UIntArray")
	add("kotlin", "java.util.List", "java.lang.Long", "LongArray", "ULongArray")
	add("kotlin", "java.util.List", "java.lang.Float", "FloatArray")
	add("kotlin", "java.util.List", "java.lang.Double", "DoubleArray")
	add("kotlin", "java.util.List", "java.lang.Boolean", "BooleanArray")
	add("kotlin", "java.util.List", "java.lang.String", "CharArray")

	// kotlin.collections.*, kotlin.sequences.*
	add("kotlin.collections", "java.util.List", "", "List", "MutableList", "MutableCollection", "Iterable")
	add("kotlin.collections", "java.util.Collection", "", "Collection")
	add("kotlin.collections", "java.util.Set", "", "Set", "MutableSet")
	add("kotlin.collections", "java.util.ArrayList", "", "ArrayList")
	add("kotlin.collections", "java.util.HashSet", "", "HashSet")
	add("kotlin.collections", "java.util.LinkedHashSet", "", "LinkedHashSet")
	add("kotlin.collections", "java.util.Map", "", "Map", "MutableMap", "HashMap", "LinkedHashMap")
	add("kotlin.sequences", "java.util.List", "", "Sequence")

	// The JDK's own spelling, written in full or imported by name.
	add("java.lang", "java.lang.Integer", "", "Integer")
	add("java.lang", "java.lang.Long", "", "Long")
	add("java.lang", "java.lang.Float", "", "Float")
	add("java.lang", "java.lang.Double", "", "Double")
	add("java.lang", "java.lang.Boolean", "", "Boolean")
	add("java.lang", "java.lang.String", "", "String")
	add("java.lang", "java.lang.Character", "", "Character")
	add("java.lang", "java.lang.Short", "", "Short")
	add("java.lang", "java.lang.Byte", "", "Byte")
	add("java.lang", "java.lang.Number", "", "Number")
	add("java.lang", "java.lang.Object", "", "Object")
	add("java.lang", "java.lang.Void", "", "Void")
	add("java.lang", "java.util.List", "", "Iterable")
	add("java.util", "java.util.List", "", "List")
	add("java.util", "java.util.Collection", "", "Collection")
	add("java.util", "java.util.Set", "", "Set")
	add("java.util", "java.util.ArrayList", "", "ArrayList")
	add("java.util", "java.util.LinkedList", "", "LinkedList")
	add("java.util", "java.util.HashSet", "", "HashSet")
	add("java.util", "java.util.LinkedHashSet", "", "LinkedHashSet")
	add("java.util", "java.util.Map", "", "Map", "HashMap", "LinkedHashMap", "TreeMap", "SortedMap")
	return types
}()

// defaultImports are the packages whose classes every Kotlin file sees without
// an import, on the JVM, from the one that wins to the one that loses.
var defaultImports = []string{"kotlin", "kotlin.collections", "kotlin.sequences", "java.lang"}

// typeScope binds the names a Kotlin file writes, as Kotlin does, so that the
// adapter hands the shared resolver a spelling that means what the Kotlin file
// meant.
//
// A name that Kotlin binds to a type of its standard library is written as the
// class of the JDK it stands for, in full. The shared resolver reads a name by
// the rules of a Java compilation unit, where `Int` is nothing and `Integer` is
// whichever class of the package, or of an import, has that name; the
// qualified name is the same in every scope.
//
// A name that Kotlin binds to a class of the project, or to a class that an
// import names, is left as written: the resolver binds it from the same
// imports and the same package.
type typeScope struct {
	// declared holds the qualified names of the project's declarations, Kotlin
	// and Java: a class of the same package, or of a package imported on demand,
	// that carries the name of a standard type hides it.
	declared map[string]bool
}

func newTypeScope(classes []ktparser.Class, javaTypes []javaparser.Class) *typeScope {
	s := &typeScope{declared: make(map[string]bool, len(classes)+len(javaTypes))}
	for _, class := range classes {
		s.declared[qualifiedName(class)] = true
	}
	for _, class := range javaTypes {
		s.declared[qualify(class.Package, class.Name)] = true
	}
	return s
}

// declares reports whether the project declares the class with the qualified
// name. A nil scope knows no declaration.
func (s *typeScope) declares(qualified string) bool {
	return s != nil && s.declared[qualified]
}

// builtin returns the standard type that a name written in unit denotes, when
// it denotes one.
//
// A name spelled with its package is that class or it is not. A simple name is
// bound in the order Kotlin binds the names of the top level of a file, which
// the Kotlin specification gives under "Call without an explicit receiver"
// (Overload resolution, 3.1 to 3.4):
//
//  1. explicit imports,
//  2. declarations of the same package,
//  3. star-imports,
//  4. implicit imports: kotlin.*, kotlin.collections.*, kotlin.sequences.* and,
//     on the JVM, java.lang.* (Packages and imports, "Implicit imports").
//
// The specification states the order for callables, a class being one through
// its constructor; the compiler binds a type name the same way.
func (s *typeScope) builtin(unit *ktparser.Class, name string) (builtinType, bool) {
	if strings.Contains(name, ".") {
		b, ok := builtinTypes[name]
		return b, ok
	}
	if unit != nil {
		for _, imp := range unit.Imports {
			if imp.OnDemand || imp.Name() != name {
				continue
			}
			if b, ok := builtinTypes[imp.Path]; ok {
				return b, true
			}
			if imp.Alias != "" {
				// Only the alias denotes the class in this file, and no
				// compilation unit of the shared resolver knows the alias: the
				// class is written by its own name.
				return builtinType{name: imp.Path}, true
			}
			return builtinType{}, false
		}
		if s.declares(qualify(unit.Package, name)) {
			return builtinType{}, false
		}
		for _, imp := range unit.Imports {
			if !imp.OnDemand {
				continue
			}
			key := imp.Path + "." + name
			if s.declares(key) {
				return builtinType{}, false
			}
			if b, ok := builtinTypes[key]; ok {
				return b, true
			}
		}
	}
	for _, pkg := range defaultImports {
		if b, ok := builtinTypes[pkg+"."+name]; ok {
			return b, true
		}
	}
	return builtinType{}, false
}

// normalize spells a type written in unit the way the shared resolver reads it.
// Standard types become the classes of the JDK they stand for, in full (see
// builtin); every other name is kept as written, and so are the arguments'.
func (s *typeScope) normalize(unit *ktparser.Class, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if raw == "*" {
		return "?"
	}
	base, args := splitGenericType(raw)
	if b, ok := s.builtin(unit, base); ok {
		base = b.name
		if args == nil && b.element != "" {
			args = []string{b.element}
		}
	}
	if args == nil {
		return base
	}
	for i := range args {
		args[i] = s.normalize(unit, args[i])
	}
	return base + "<" + strings.Join(args, ",") + ">"
}

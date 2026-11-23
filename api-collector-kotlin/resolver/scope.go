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
	add("kotlin", "java.lang.CharSequence", "", "CharSequence")
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
	add("java.lang", "java.lang.CharSequence", "", "CharSequence")
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
// A name that Kotlin binds to a typealias is replaced by the type the alias
// stands for, read in the file that declares the alias.
//
// A name that Kotlin binds to a class of the project, or to a class that an
// import names, is left as written: the resolver binds it from the same
// imports and the same package.
type typeScope struct {
	// top holds the top-level declarations of the project, Kotlin, Java and
	// typealias, by qualified name: one of the same package, or of a package
	// imported on demand, that carries the name of a standard type hides it. A
	// nested class is not among them: it is in scope only around its outer class
	// (see memberType).
	top map[string][]declaration
	// members holds the qualified name, outer classes included, of every nested
	// class of the project.
	members map[string]bool
	// classes holds the Kotlin classes, nested ones too, by that same name, for
	// the supertypes a nested class is inherited through, and javaClasses the
	// Java ones: a Kotlin class inherits the classes nested in its Java
	// superclass as well.
	classes     map[string]*ktparser.Class
	javaClasses map[string]*javaparser.Class
	// aliases holds the typealiases by qualified name.
	aliases map[string][]*ktparser.TypeAlias
}

// declaration is one place a top-level name is declared.
type declaration struct {
	file string
	// private marks a Kotlin declaration only its own file sees.
	private bool
}

func newTypeScope(classes []ktparser.Class, javaTypes []javaparser.Class) *typeScope {
	s := &typeScope{
		top:     make(map[string][]declaration, len(classes)+len(javaTypes)),
		members: make(map[string]bool),
		classes: make(map[string]*ktparser.Class, len(classes)),
		aliases: make(map[string][]*ktparser.TypeAlias),

		javaClasses: make(map[string]*javaparser.Class, len(javaTypes)),
	}
	for i := range classes {
		class := &classes[i]
		key := qualifiedName(*class)
		s.classes[key] = class
		if class.Outer != "" {
			s.members[key] = true
			continue
		}
		s.top[key] = append(s.top[key], declaration{file: class.FilePath, private: class.IsPrivate})
	}
	for i := range javaTypes {
		class := &javaTypes[i]
		name := class.Name
		if class.Outer != "" {
			name = class.Outer + "." + name
		}
		key := qualify(class.Package, name)
		s.javaClasses[key] = class
		if class.Outer != "" {
			s.members[key] = true
			continue
		}
		s.top[key] = append(s.top[key], declaration{file: class.FilePath})
	}
	return s
}

// addAliases makes the typealiases of the project part of the scope.
func (s *typeScope) addAliases(aliases []ktparser.TypeAlias) {
	for i := range aliases {
		alias := &aliases[i]
		key := qualify(alias.Package, alias.Name)
		s.aliases[key] = append(s.aliases[key], alias)
		s.top[key] = append(s.top[key], declaration{file: alias.FilePath, private: alias.IsPrivate})
	}
}

// declares reports whether the project declares a top-level class or alias with
// the qualified name that unit can see. A nil scope knows no declaration.
func (s *typeScope) declares(unit *ktparser.Class, qualified string) bool {
	if s == nil {
		return false
	}
	for _, d := range s.top[qualified] {
		if !d.private || (unit != nil && d.file == unit.FilePath) {
			return true
		}
	}
	return false
}

// isMember reports whether the qualified name is that of a class nested in
// another one.
func (s *typeScope) isMember(qualified string) bool {
	return s != nil && s.members[qualified]
}

// aliasAt returns the typealias with the qualified name that unit can see.
func (s *typeScope) aliasAt(unit *ktparser.Class, qualified string) *ktparser.TypeAlias {
	if s == nil {
		return nil
	}
	for _, alias := range s.aliases[qualified] {
		if !alias.IsPrivate || (unit != nil && alias.FilePath == unit.FilePath) {
			return alias
		}
	}
	return nil
}

// maxSuperDepth bounds the walk up the supertypes of a class, so a cycle cannot
// loop.
const maxSuperDepth = 8

// memberType reports whether name is a nested class that the body of unit sees
// by its simple name: one nested in unit, in a class that encloses it, or in a
// supertype of either. Kotlin looks for it there before it looks at the file
// and the package.
func (s *typeScope) memberType(unit *ktparser.Class, name string) bool {
	if s == nil || unit == nil || unit.Name == "" {
		return false
	}
	seen := map[string]bool{}
	var visit func(key string, depth int) bool
	visit = func(key string, depth int) bool {
		if seen[key] || depth > maxSuperDepth {
			return false
		}
		seen[key] = true
		if s.members[key+"."+name] {
			return true
		}
		scope, supers, ok := s.supertypesOf(key)
		if !ok {
			return false
		}
		for _, super := range supers {
			if super == "" {
				continue
			}
			if superKey, ok := s.superKey(scope, super); ok && visit(superKey, depth+1) {
				return true
			}
		}
		return false
	}
	if visit(qualifiedName(*unit), 0) {
		return true
	}
	for outer := unit.Outer; outer != ""; {
		if visit(qualify(unit.Package, outer), 0) {
			return true
		}
		i := strings.LastIndex(outer, ".")
		if i < 0 {
			break
		}
		outer = outer[:i]
	}
	return false
}

// importRef is an import of a compilation unit, of either language, as the
// lookup of a supertype reads it.
type importRef struct {
	path     string
	onDemand bool
	// name is the simple name the import binds; a wildcard binds none.
	name string
}

// declaringScope is the scope a class writes its supertypes in: its package, the
// classes around it and its imports.
type declaringScope struct {
	pkg, outer string
	imports    []importRef
}

// supertypesOf returns the scope and the supertypes written by the class held
// under key, of Kotlin or of Java.
func (s *typeScope) supertypesOf(key string) (declaringScope, []string, bool) {
	if class := s.classes[key]; class != nil {
		scope := declaringScope{pkg: class.Package, outer: class.Outer}
		for _, imp := range class.Imports {
			scope.imports = append(scope.imports, importRef{path: imp.Path, onDemand: imp.OnDemand, name: imp.Name()})
		}
		return scope, append([]string{class.SuperClass}, class.Interfaces...), true
	}
	if class := s.javaClasses[key]; class != nil {
		scope := declaringScope{pkg: class.Package, outer: class.Outer}
		for _, imp := range class.Imports {
			if imp.Static {
				continue
			}
			scope.imports = append(scope.imports, importRef{path: imp.Path, onDemand: imp.OnDemand, name: lastSegment(imp.Path)})
		}
		return scope, append([]string{class.SuperClass}, class.Interfaces...), true
	}
	return declaringScope{}, nil, false
}

// holds reports whether a class of the project stands under key, in either
// language.
func (s *typeScope) holds(key string) bool {
	return s.classes[key] != nil || s.javaClasses[key] != nil
}

// superKey returns the qualified name of the class of the project that a
// supertype written in scope names.
func (s *typeScope) superKey(scope declaringScope, raw string) (string, bool) {
	name, _ := splitGenericType(raw)
	if name == "" {
		return "", false
	}
	known := func(key string) (string, bool) {
		return key, s.holds(key)
	}
	if strings.Contains(name, ".") {
		if key, ok := known(qualify(scope.pkg, name)); ok {
			return key, true
		}
		return known(name)
	}
	for _, imp := range scope.imports {
		if !imp.onDemand && imp.name == name {
			return known(imp.path)
		}
	}
	for outer := scope.outer; outer != ""; {
		if key, ok := known(qualify(scope.pkg, outer+"."+name)); ok {
			return key, true
		}
		i := strings.LastIndex(outer, ".")
		if i < 0 {
			break
		}
		outer = outer[:i]
	}
	if key, ok := known(qualify(scope.pkg, name)); ok {
		return key, true
	}
	for _, imp := range scope.imports {
		if imp.onDemand {
			if key, ok := known(imp.path + "." + name); ok {
				return key, true
			}
		}
	}
	return "", false
}

// binding is what a name written in a file denotes, when it denotes a standard
// type or a typealias; a name that denotes neither has the zero value.
type binding struct {
	builtin *builtinType
	alias   *ktparser.TypeAlias
}

// bind finds what a name written in unit denotes.
//
// A name spelled with its package is that class or it is not. A simple name is
// bound in the order Kotlin binds the names of a class body and the top level of
// a file, which the Kotlin specification gives under "Call without an explicit
// receiver" (Overload resolution, 3.1 to 3.4):
//
//  0. nested classes in scope in the body: those of the class, of the classes
//     around it and of their supertypes,
//  1. explicit imports,
//  2. declarations of the same package,
//  3. star-imports,
//  4. implicit imports: kotlin.*, kotlin.collections.*, kotlin.sequences.* and,
//     on the JVM, java.lang.* (Packages and imports, "Implicit imports").
//
// The specification states the order for callables, a class being one through
// its constructor; the compiler binds a type name the same way. A private
// top-level declaration of another file is not in the package's scope.
func (s *typeScope) bind(unit *ktparser.Class, name string) binding {
	if strings.Contains(name, ".") {
		if b, ok := builtinTypes[name]; ok {
			return binding{builtin: &b}
		}
		return binding{alias: s.aliasAt(unit, name)}
	}
	if unit != nil {
		if s.memberType(unit, name) {
			return binding{}
		}
		for _, imp := range unit.Imports {
			if imp.OnDemand || imp.Name() != name {
				continue
			}
			if b, ok := builtinTypes[imp.Path]; ok {
				return binding{builtin: &b}
			}
			if alias := s.aliasAt(unit, imp.Path); alias != nil {
				return binding{alias: alias}
			}
			if imp.Alias != "" {
				// Only the alias denotes the class in this file, and no
				// compilation unit of the shared resolver knows the alias: the
				// class is written by its own name.
				return binding{builtin: &builtinType{name: imp.Path}}
			}
			return binding{}
		}
		key := qualify(unit.Package, name)
		if alias := s.aliasAt(unit, key); alias != nil {
			return binding{alias: alias}
		}
		if s.declares(unit, key) {
			return binding{}
		}
		for _, imp := range unit.Imports {
			if !imp.OnDemand {
				continue
			}
			key := imp.Path + "." + name
			if alias := s.aliasAt(unit, key); alias != nil {
				return binding{alias: alias}
			}
			// A star-import of a class brings in the classes nested in it.
			if s.declares(unit, key) || s.isMember(key) {
				return binding{}
			}
			if b, ok := builtinTypes[key]; ok {
				return binding{builtin: &b}
			}
		}
	}
	for _, pkg := range defaultImports {
		if b, ok := builtinTypes[pkg+"."+name]; ok {
			return binding{builtin: &b}
		}
	}
	return binding{}
}

// spelling carries what a type is being written under: the type parameters of
// an alias that are replaced by the arguments it was used with, how deep in
// aliases the walk is, and whether the classes of the project are named in
// full.
type spelling struct {
	subst   map[string]string
	depth   int
	qualify bool
}

// maxAliasDepth bounds a chain of typealiases defined by one another, so a cycle
// cannot loop.
const maxAliasDepth = 8

// normalize spells a type written in unit the way the shared resolver reads it.
// Standard types become the classes of the JDK they stand for, in full, and a
// typealias becomes the type it names (see bind); every other name is kept as
// written, and so are the arguments'.
func (s *typeScope) normalize(unit *ktparser.Class, raw string) string {
	return s.spell(unit, raw, spelling{})
}

func (s *typeScope) spell(unit *ktparser.Class, raw string, sp spelling) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if raw == "*" {
		return "?"
	}
	base, args := splitGenericType(raw)
	if args == nil {
		if replacement, ok := sp.subst[base]; ok {
			return replacement
		}
	}
	b := s.bind(unit, base)
	switch {
	case b.alias != nil:
		if sp.depth >= maxAliasDepth || strings.Contains(b.alias.Target, "->") {
			// A cycle, or a function type that no class stands for: the alias
			// is written by its own name, which nothing resolves.
			return qualify(b.alias.Package, b.alias.Name)
		}
		return s.expand(unit, b.alias, args, sp)
	case b.builtin != nil:
		base = b.builtin.name
		if args == nil && b.builtin.element != "" {
			args = []string{b.builtin.element}
		}
	case sp.qualify:
		base = s.qualifyProject(unit, base)
	}
	if args == nil {
		return base
	}
	for i := range args {
		args[i] = s.spell(unit, args[i], sp)
	}
	return base + "<" + strings.Join(args, ",") + ">"
}

// expand writes the type that an alias names, read in the file that declares the
// alias. The arguments the alias is used with replace its type parameters; they
// were written where the alias is used, so they are read there.
func (s *typeScope) expand(use *ktparser.Class, alias *ktparser.TypeAlias, args []string, sp spelling) string {
	subst := make(map[string]string, len(alias.TypeParameters))
	for i, param := range alias.TypeParameters {
		if i < len(args) {
			subst[param] = s.spell(use, args[i], sp)
		}
	}
	declaring := &ktparser.Class{Package: alias.Package, FilePath: alias.FilePath, Imports: alias.Imports}
	return s.spell(declaring, alias.Target, spelling{subst: subst, depth: sp.depth + 1, qualify: true})
}

// qualifyProject writes a class of the project that unit names by its simple
// name in full, so that it keeps meaning that class where the type is read
// next. A name that no declaration of the project and no import gives is left
// as written.
func (s *typeScope) qualifyProject(unit *ktparser.Class, name string) string {
	if unit == nil || strings.Contains(name, ".") {
		return name
	}
	for _, imp := range unit.Imports {
		if !imp.OnDemand && imp.Name() == name {
			return imp.Path
		}
	}
	if key := qualify(unit.Package, name); s.declares(unit, key) {
		return key
	}
	for _, imp := range unit.Imports {
		if imp.OnDemand {
			if key := imp.Path + "." + name; s.declares(unit, key) || s.isMember(key) {
				return key
			}
		}
	}
	return name
}

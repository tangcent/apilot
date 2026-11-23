package resolver

import (
	"strconv"
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// maxConstantHops bounds a chain of constants defined by one another
// (VERSION = Base.VERSION), so a cycle cannot loop.
const maxConstantHops = 8

// ConstantValue returns the value of the String constant that ref names as
// written in source: VERSION, Headers.VERSION or com.example.Headers.VERSION.
//
// The compiler substitutes a constant for its value, so an annotation
// attribute such as @RequestHeader(Headers.VERSION) means the string the
// constant holds. The name is looked up the way the compiler does: a simple
// name among the fields of source and its supertypes, then through the single
// and on-demand static imports; a qualified name in the class its qualifier
// denotes, which is resolved like a type name. A Kotlin member import
// (import com.example.Headers.VERSION) has the shape of a single static
// import, and a const val of a Kotlin companion object or object is a static
// field of its class, so Kotlin declarations converted to this model resolve
// the same way. A class missing from the scanned sources is read from the
// project's dependencies when a dependency resolver is set.
//
// ok is false when the constant, or the class declaring it, cannot be found,
// or when its initializer is not a string literal or the name of another
// constant.
func (r *TypeResolver) ConstantValue(source parser.Class, ref string) (string, bool) {
	return r.constantValue(source, strings.Join(strings.Fields(ref), ""), 0)
}

func (r *TypeResolver) constantValue(source parser.Class, ref string, hops int) (string, bool) {
	if ref == "" || hops > maxConstantHops {
		return "", false
	}
	dot := strings.LastIndex(ref, ".")
	if dot < 0 {
		return r.simpleConstant(source, ref, hops)
	}
	owner, member := ref[:dot], ref[dot+1:]
	// Kotlin also spells a companion member Headers.Companion.VERSION; the
	// constant is a static field of Headers either way.
	if strings.HasSuffix(owner, ".Companion") {
		owner = strings.TrimSuffix(owner, ".Companion")
	}
	class, found := r.constantClass(source, owner)
	if !found {
		return "", false
	}
	return r.memberConstant(class, member, hops, map[string]bool{})
}

// simpleConstant looks a constant written by its simple name up from source.
func (r *TypeResolver) simpleConstant(source parser.Class, name string, hops int) (string, bool) {
	if value, found := r.memberConstant(source, name, hops, map[string]bool{}); found {
		return value, true
	}
	// A single import of the member: import static com.example.Headers.VERSION
	// in Java, import com.example.Headers.VERSION in Kotlin.
	for _, imp := range source.Imports {
		if imp.OnDemand || lastSegment(imp.Path) != name || !strings.Contains(imp.Path, ".") {
			continue
		}
		owner := imp.Path[:strings.LastIndex(imp.Path, ".")]
		if class, found := r.constantClass(source, owner); found {
			if value, found := r.memberConstant(class, name, hops, map[string]bool{}); found {
				return value, true
			}
		}
	}
	for _, imp := range source.Imports {
		if !imp.Static || !imp.OnDemand {
			continue
		}
		if class, found := r.constantClass(source, imp.Path); found {
			if value, found := r.memberConstant(class, name, hops, map[string]bool{}); found {
				return value, true
			}
		}
	}
	return "", false
}

// memberConstant returns the value of the constant named name that class
// declares or inherits from a superclass or an interface.
func (r *TypeResolver) memberConstant(class parser.Class, name string, hops int, visited map[string]bool) (string, bool) {
	key := qualifiedName(class)
	if visited[key] {
		return "", false
	}
	visited[key] = true

	for _, f := range class.Fields {
		if f.Name != name || f.Constant == "" {
			continue
		}
		if strings.HasPrefix(f.Constant, `"`) {
			return unquoteStringLiteral(f.Constant), true
		}
		// The initializer names another constant, as written in the
		// compilation unit that declares this one.
		return r.constantValue(class, f.Constant, hops+1)
	}

	supertypes := append([]string{class.SuperClass}, class.Interfaces...)
	for _, super := range supertypes {
		base, _ := parseGenericType(strings.TrimSpace(super))
		if base == "" {
			continue
		}
		if superClass, found := r.constantClass(class, base); found {
			if value, found := r.memberConstant(superClass, name, hops, visited); found {
				return value, true
			}
		}
	}
	return "", false
}

// constantClass returns the class a qualifier denotes as seen from source: a
// simple name resolved like a type name, a class nested in a type in scope
// (Outer.Headers), or a qualified name. A class missing from the scanned
// sources is read from the dependencies.
func (r *TypeResolver) constantClass(source parser.Class, name string) (parser.Class, bool) {
	key := r.resolveName(source, name)
	if c, found := r.classRegistry[key]; found {
		return c, true
	}
	// Only a qualified name can be asked of the dependencies, and a package of
	// the scanned sources is not a class: import com.example.VERSION imports a
	// Kotlin top-level constant, not a member of a class named example. The
	// JDK and the Kotlin standard library are not asked either: they are
	// reached as supertypes (Serializable) far more often than as the home of
	// a binding name.
	if r.dependencyResolver == nil || !strings.Contains(key, ".") || isPlatformName(key) || r.isPackage(key) || r.memberOfScanned(key) {
		return parser.Class{}, false
	}
	if c := r.readDependency(key); c != nil {
		return *c, true
	}
	return parser.Class{}, false
}

// isPackage reports whether name is the package of a declaration the resolver
// holds.
func (r *TypeResolver) isPackage(name string) bool {
	for _, c := range r.classRegistry {
		if c.Package == name {
			return true
		}
	}
	return false
}

func qualify(pkg, name string) string {
	if pkg == "" {
		return name
	}
	return pkg + "." + name
}

// unquoteStringLiteral returns the string a Java or Kotlin string literal
// spells. A Kotlin raw string ("""...""") has no escapes; an escape Go does not
// know, such as Kotlin's \$, falls back to the text between the quotes with
// that escape undone.
func unquoteStringLiteral(literal string) string {
	if len(literal) >= 6 && strings.HasPrefix(literal, `"""`) && strings.HasSuffix(literal, `"""`) {
		return literal[3 : len(literal)-3]
	}
	if value, err := strconv.Unquote(literal); err == nil {
		return value
	}
	if value, err := strconv.Unquote(strings.ReplaceAll(literal, `\$`, `$`)); err == nil {
		return value
	}
	return strings.TrimSuffix(strings.TrimPrefix(literal, `"`), `"`)
}

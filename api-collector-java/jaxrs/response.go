package jaxrs

import (
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// JAX-RS Response is opaque: the declared type names no payload, and the
// payload is built inside the method body (`Response.ok(entity)`). Reading
// only the return type loses the entire response section, because the wrapper
// resolves to nothing at all.
//
// The helpers below recover the entity from the body text. They are plain text
// scanning rather than a second AST pass: the body is short, the shapes are few
// and regular, and a receiver or argument that cannot be named is reported as
// opaque instead of guessed at.

// isJaxrsResponse reports whether returnType denotes the JAX-RS Response type,
// in either namespace: javax.ws.rs until Jakarta EE 9, jakarta.ws.rs after it.
func isJaxrsResponse(returnType string) bool {
	switch strings.TrimSpace(returnType) {
	case "Response", "javax.ws.rs.core.Response", "jakarta.ws.rs.core.Response":
		return true
	}
	return false
}

// responsePayload recovers the payload type of a JAX-RS Response from the
// method body, where the entity is built rather than declared.
//
// body is the raw method body, params maps the method's parameter names to
// their declared types, and classRegistry supplies the type-parameter counts
// that turn `Result.success(category)` into `Result<Category>`.
//
// The result is one of three states:
//
//	payload != ""      the entity type, ready for the type resolver
//	payload == "", !opaque  the body builds a Response with no entity
//	                        (`Response.noContent().build()`): nothing is lost
//	payload == "", opaque   a payload was built that this inspection cannot
//	                        name; the caller must say so out loud
//
// Every entity these heuristics can name is in the body itself — a constructor
// call, a local variable, a method parameter, or a static factory such as
// `Result.success(x)`. Anything else (a value returned by another method, a
// variable assembled across branches) is reported as opaque instead of being
// guessed.
func responsePayload(body string, params map[string]string, classRegistry map[string]parser.Class) (string, bool) {
	expr, found := responseEntityExpr(body)
	if !found {
		// No Response construction at all: the value is produced elsewhere
		// (a helper call, a variable assigned in several branches).
		return "", true
	}
	if expr == "" {
		// A construction with no entity. The body genuinely has no payload,
		// which is not a gap worth a warning.
		return "", false
	}
	if typeName, ok := entityType(expr, params, body, classRegistry); ok {
		return typeName, false
	}
	return "", true
}

// responseEntityExpr finds the entity a Response is built with: the earliest
// entity-bearing construction in the body, since a method that attaches an
// entity anywhere in its branches is documenting that payload.
//
// found is false when the body contains no recognised Response construction.
// An empty expr with found true means the constructions carry no entity.
func responseEntityExpr(body string) (expr string, found bool) {
	// `.entity(x)` is the explicit builder call and can appear on any
	// ResponseBuilder, including one obtained from Response.status(...).
	if idx := strings.Index(body, ".entity("); idx >= 0 {
		found = true
		if arg, ok := firstCallArgument(body[idx:], ".entity(", 0); ok && arg != "" {
			return arg, true
		}
	}

	// The factory methods that take an entity. Positions follow the JAX-RS
	// API: ok/accepted/serverError take it first, status/created after their
	// leading status or location argument. A call found without an argument at
	// that position (`ok()`, `status(204)`) is a construction with no entity.
	entities := []struct {
		method string
		index  int
	}{
		{"Response.ok(", 0},
		{"Response.accepted(", 0},
		{"Response.serverError(", 0},
		{"Response.status(", 1},
		{"Response.created(", 1},
	}

	earliest := -1
	for _, candidate := range entities {
		idx := strings.Index(body, candidate.method)
		if idx < 0 {
			continue
		}
		found = true
		arg, ok := firstCallArgument(body[idx:], candidate.method, candidate.index)
		if !ok || arg == "" {
			continue
		}
		if earliest < 0 || idx < earliest {
			earliest, expr = idx, arg
		}
	}
	if earliest >= 0 {
		return expr, true
	}

	// Factories that never carry an entity. They are recognised so that a
	// `Response.noContent().build()` body reports "no payload" rather than the
	// payload it never had being called unrecoverable.
	for _, call := range []string{
		"Response.noContent(",
		"Response.notModified(",
		"Response.seeOther(",
		"Response.temporaryRedirect(",
	} {
		if strings.Contains(body, call) {
			found = true
		}
	}

	return "", found
}

// firstCallArgument returns the argument at index of the first occurrence of
// call in s. ok reports whether the call was found at all.
//
// Multi-line chains are common (`Response\n.status(201)\n.entity(x)`), so the
// argument list is read by matching parentheses rather than by line.
func firstCallArgument(s, call string, index int) (string, bool) {
	idx := strings.Index(s, call)
	if idx < 0 {
		return "", false
	}
	args, ok := callArguments(s[idx+len(call)-1:])
	if !ok {
		return "", false
	}
	if index >= len(args) {
		return "", true
	}
	return args[index], true
}

// callArguments reads the argument list that starts at the opening parenthesis
// at s[0] and returns its top-level arguments, trimmed. A nil argument with ok
// true means the list is empty.
func callArguments(s string) ([]string, bool) {
	if s == "" || s[0] != '(' {
		return nil, false
	}

	var args []string
	var current strings.Builder
	depth := 0
	angle := 0

	for i := 1; i < len(s); i++ {
		c := s[i]
		switch c {
		case '(', '[', '{':
			depth++
		case ')':
			if depth == 0 {
				args = append(args, strings.TrimSpace(current.String()))
				return trimEmptyEdges(args), true
			}
			depth--
		case ']', '}':
			if depth > 0 {
				depth--
			}
		case '<':
			angle++
		case '>':
			if angle > 0 {
				angle--
			}
		case ',':
			if depth == 0 && angle == 0 {
				args = append(args, strings.TrimSpace(current.String()))
				current.Reset()
				continue
			}
		case '"', '\'':
			end := strings.IndexByte(s[i+1:], c)
			if end < 0 {
				return nil, false
			}
			current.WriteString(s[i : i+end+2])
			i += end + 1
			continue
		}
		current.WriteByte(c)
	}
	return nil, false
}

// trimEmptyEdges drops the blank argument an empty list produces, so `()` and
// `( )` both come back as no arguments rather than one empty one.
func trimEmptyEdges(args []string) []string {
	if len(args) == 1 && args[0] == "" {
		return nil
	}
	return args
}

// entityType reduces an entity expression to the type it carries.
func entityType(expr string, params map[string]string, body string, classRegistry map[string]parser.Class) (string, bool) {
	expr = strings.TrimSpace(expr)

	if typeName, ok := newExpressionType(expr); ok {
		return typeName, true
	}

	// A method parameter, then a local variable. Java forbids one from
	// shadowing the other, so the order does not matter.
	if isIdentifier(expr) {
		if typeName, ok := params[expr]; ok && typeName != "" {
			return typeName, true
		}
		return localVariableType(body, expr)
	}

	// A static factory: `Result.success(category)` carries a Result, whose
	// type parameter comes from the argument. This is the shape that makes a
	// wrapped export useful, so it is worth the one inference step.
	if base, args, ok := staticFactoryCall(expr); ok {
		if typeParameterCount(classRegistry, base) == 1 && len(args) == 1 {
			if argType, ok := entityType(args[0], params, body, classRegistry); ok {
				return base + "<" + argType + ">", true
			}
		}
		return base, true
	}

	return "", false
}

// newExpressionType reduces `new Order(...)` to `Order`, and `new Page<T>(...)`
// to `Page<T>`. A diamond (`new Page<>(...)`) leaves the argument types to the
// compiler, so it reduces to `Page`.
func newExpressionType(expr string) (string, bool) {
	const keyword = "new "
	if !strings.HasPrefix(expr, keyword) {
		return "", false
	}

	rest := strings.TrimSpace(expr[len(keyword):])
	end := 0
	depth := 0
	for end < len(rest) {
		c := rest[end]
		if c == '<' {
			depth++
		} else if c == '>' {
			depth--
			if depth == 0 {
				end++
				break
			}
		} else if depth == 0 && !isTypeChar(c) {
			break
		}
		end++
	}

	typeName := simpleTypeName(rest[:end])
	if typeName == "" || !isClassName(typeName) {
		return "", false
	}
	return typeName, true
}

// staticFactoryCall splits `Result.success(category)` into `Result` and its
// arguments. Only an uppercase receiver qualifies: `service.load(id)` is an
// instance call and its receiver is not a type.
func staticFactoryCall(expr string) (string, []string, bool) {
	open := strings.IndexByte(expr, '(')
	if open < 0 {
		return "", nil, false
	}
	callee := strings.TrimSpace(expr[:open])
	dot := strings.LastIndexByte(callee, '.')
	if dot <= 0 {
		return "", nil, false
	}
	base := simpleTypeName(callee[:dot])
	if !isClassName(base) {
		return "", nil, false
	}
	if !isIdentifier(callee[dot+1:]) {
		return "", nil, false
	}
	args, ok := callArguments(expr[open:])
	if !ok {
		return "", nil, false
	}
	return base, args, true
}

// localVariableType finds the declared type of name inside the method body by
// looking for `<Type> name = ...`.
func localVariableType(body, name string) (string, bool) {
	for i := 0; i+len(name) <= len(body); i++ {
		if !strings.HasPrefix(body[i:], name) {
			continue
		}
		if i > 0 && isIdentChar(body[i-1]) {
			continue
		}
		after := i + len(name)
		if after < len(body) && isIdentChar(body[after]) {
			continue
		}

		eq := after
		for eq < len(body) && (body[eq] == ' ' || body[eq] == '\t') {
			eq++
		}
		// `==` and `!=` are comparisons, not the initialiser this is after.
		if eq >= len(body) || body[eq] != '=' {
			continue
		}
		if eq+1 < len(body) && body[eq+1] == '=' {
			continue
		}

		typeName, ok := typeBefore(body[:i])
		if !ok {
			continue
		}
		return typeName, true
	}
	return "", false
}

// typeBefore reads the type token that ends just before a declarator name,
// skipping trailing space. Nested type arguments are skipped as a unit.
func typeBefore(prefix string) (string, bool) {
	i := len(prefix) - 1
	for i >= 0 && (prefix[i] == ' ' || prefix[i] == '\t') {
		i--
	}
	if i < 0 {
		return "", false
	}

	end := i
	for i >= 0 {
		c := prefix[i]
		switch {
		case c == '>':
			depth := 0
			for i >= 0 {
				if prefix[i] == '>' {
					depth++
				} else if prefix[i] == '<' {
					depth--
					if depth == 0 {
						i--
						break
					}
				}
				i--
			}
		case isIdentChar(c) || c == '.':
			i--
		default:
			goto done
		}
	}

done:
	typeName := simpleTypeName(strings.TrimSpace(prefix[i+1 : end+1]))
	if !isClassName(typeName) {
		return "", false
	}
	return typeName, true
}

// simpleTypeName drops a package qualifier but keeps type arguments:
// `com.example.Result<Order>` becomes `Result<Order>`.
func simpleTypeName(typeName string) string {
	typeName = strings.TrimSpace(typeName)
	if typeName == "" {
		return ""
	}
	name := typeName
	args := ""
	if idx := strings.IndexByte(typeName, '<'); idx >= 0 {
		name = typeName[:idx]
		args = typeName[idx:]
	}
	if args == "<>" {
		args = ""
	}
	if idx := strings.LastIndexByte(name, '.'); idx >= 0 {
		name = name[idx+1:]
	}
	return name + args
}

// isClassName reports whether name can name a Java type. A local variable or
// an instance is camelCase, so requiring an uppercase first letter keeps
// `service.load(id)` from being read as the type `service`.
func isClassName(name string) bool {
	if name == "" {
		return false
	}
	first := name[0]
	return first >= 'A' && first <= 'Z'
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isIdentChar(c) {
			continue
		}
		return false
	}
	first := s[0]
	return !(first >= '0' && first <= '9')
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func isIdentChar(c byte) bool {
	return c == '_' || c == '$' ||
		(c >= 'a' && c <= 'z') ||
		(c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9')
}

func isTypeChar(c byte) bool {
	return isIdentChar(c) || c == '.' || c == '<' || c == '>' || c == ',' || c == '[' || c == ']'
}

// typeParameterCount reports how many type parameters a class declares, and 0
// when the class is not in the registry. It is what makes
// `Result.success(category)` bind to `Result<Category>` instead of staying raw.
func typeParameterCount(classRegistry map[string]parser.Class, name string) int {
	class, found := classRegistry[name]
	if !found {
		return 0
	}
	return len(class.TypeParameters)
}

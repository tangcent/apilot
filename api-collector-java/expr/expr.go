// Package expr reads the type a Java expression carries from its source text.
//
// A framework parser needs this where the payload is not named in the method
// signature: a JAX-RS Response whose entity is built in the body, or a
// wildcard type argument that only the body fills in. Method bodies are short
// and the shapes are few and regular, so plain text scanning is enough; an
// expression this package cannot name is reported as unknown rather than
// guessed at.
package expr

import (
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// Lookup returns the declaration of a type name as seen from the compilation
// unit being exported.
//
// Payload inspection asks the same question the type resolver does: whether
// `Result.success(x)` is `Result<T>` depends on which Result the file can see,
// so the answer comes from the resolver instead of a simple-name registry of
// its own.
type Lookup func(name string) (parser.Class, bool)

// ReturnedType reduces the return expressions of a method body and reports the
// type they agree on.
//
// Expressions that name no type are skipped. Two returns naming different
// types are conflicting evidence and are not resolved: the caller keeps the
// declared type instead of documenting one branch as if it were the method's
// answer.
func ReturnedType(method parser.Method, lookup Lookup) (string, bool) {
	params := make(map[string]string, len(method.Parameters))
	for _, param := range method.Parameters {
		if param.Name != "" && param.Type != "" {
			params[param.Name] = param.Type
		}
	}

	var agreed string
	var found bool
	for _, returned := range returnedExpressions(method.Body) {
		typeName, ok := Type(returned, params, method.Body, lookup)
		if !ok {
			continue
		}
		if found && typeName != agreed {
			return "", false
		}
		agreed, found = typeName, true
	}
	return agreed, found
}

// Type reduces an expression to the type it carries.
//
// Every type this can name is in the expression itself — a constructor call, a
// local variable, a method parameter, or a static factory such as
// `Result.success(x)`. Anything else (a value returned by another method, a
// variable assembled across branches) is reported as unknown.
func Type(expression string, params map[string]string, body string, lookup Lookup) (string, bool) {
	expression = strings.TrimSpace(expression)

	if typeName, ok := newExpressionType(expression); ok {
		return typeName, true
	}

	// A method parameter, then a local variable. Java forbids one from
	// shadowing the other, so the order does not matter.
	if isIdentifier(expression) {
		if typeName, ok := params[expression]; ok && typeName != "" {
			return typeName, true
		}
		return localVariableType(body, expression)
	}

	// A static factory: `Result.success(category)` carries a Result, whose
	// type parameter comes from the argument. This is the shape that makes a
	// wrapped payload useful, so it is worth the one inference step.
	if base, args, ok := staticFactoryCall(expression); ok {
		if typeParameterCount(lookup, base) == 1 && len(args) == 1 {
			if argType, ok := Type(args[0], params, body, lookup); ok {
				return base + "<" + argType + ">", true
			}
		}
		return base, true
	}

	return "", false
}

// FirstCallArgument returns the argument at index of the first occurrence of
// call in s. ok reports whether the call was found at all.
//
// Multi-line chains are common (`Response\n.status(201)\n.entity(x)`), so the
// argument list is read by matching parentheses rather than by line.
func FirstCallArgument(s, call string, index int) (string, bool) {
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

// returnedExpressions returns the expressions of the return statements in a
// method body, in source order.
func returnedExpressions(body string) []string {
	const keyword = "return"

	var exprs []string
	for i := 0; i < len(body); i++ {
		if !strings.HasPrefix(body[i:], keyword) {
			continue
		}
		if i > 0 && isIdentChar(body[i-1]) {
			continue
		}
		expression, end, ok := expressionUntilSemicolon(body, i+len(keyword))
		if !ok {
			continue
		}
		if expression != "" {
			exprs = append(exprs, expression)
		}
		i = end
	}
	return exprs
}

// expressionUntilSemicolon reads the expression that starts at start and ends
// at the next semicolon outside any brackets. A closing brace before that
// semicolon ends the block, so the scan reports no expression.
func expressionUntilSemicolon(body string, start int) (string, int, bool) {
	var current strings.Builder
	depth := 0

	for i := start; i < len(body); i++ {
		c := body[i]
		switch c {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			if depth == 0 {
				return "", i, false
			}
			depth--
		case ';':
			if depth == 0 {
				return strings.TrimSpace(current.String()), i, true
			}
		case '"', '\'':
			end := strings.IndexByte(body[i+1:], c)
			if end < 0 {
				return "", i, false
			}
			current.WriteString(body[i : i+end+2])
			i += end + 1
			continue
		}
		current.WriteByte(c)
	}
	return "", len(body), false
}

// newExpressionType reduces `new Order(...)` to `Order`, and `new Page<T>(...)`
// to `Page<T>`. A diamond (`new Page<>(...)`) leaves the argument types to the
// compiler, so it reduces to `Page`.
func newExpressionType(expression string) (string, bool) {
	const keyword = "new "
	if !strings.HasPrefix(expression, keyword) {
		return "", false
	}

	rest := strings.TrimSpace(expression[len(keyword):])
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
func staticFactoryCall(expression string) (string, []string, bool) {
	open := strings.IndexByte(expression, '(')
	if open < 0 {
		return "", nil, false
	}
	callee := strings.TrimSpace(expression[:open])
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
	args, ok := callArguments(expression[open:])
	if !ok {
		return "", nil, false
	}
	return base, args, true
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
// when the class is not among the scanned sources. It is what makes
// `Result.success(category)` bind to `Result<Category>` instead of staying raw.
func typeParameterCount(lookup Lookup, name string) int {
	class, found := lookup(name)
	if !found {
		return 0
	}
	return len(class.TypeParameters)
}

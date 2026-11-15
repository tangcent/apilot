package parser

import (
	"fmt"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// argumentsRead lists the annotations whose arguments the collector reads:
// request mappings and bindings, Jackson naming, and the documentation
// annotations. Every other annotation is only ever checked for presence
// (@NotNull, @Valid) or ignored, so its arguments carry nothing the export
// uses.
var argumentsRead = map[string]bool{
	"RequestMapping": true, "GetMapping": true, "PostMapping": true, "PutMapping": true,
	"DeleteMapping": true, "PatchMapping": true,
	"RequestParam": true, "PathVariable": true, "RequestHeader": true, "CookieValue": true,
	"RequestPart": true, "RequestBody": true,
	"RestController": true, "Controller": true,
	"JsonProperty": true, "JsonIgnore": true,
	"ApiOperation": true, "ApiParam": true, "ApiModelProperty": true,
	"Operation": true, "Parameter": true, "Schema": true,
}

// IsControllerAnnotation reports whether an annotation, by the simple name the
// parser records for it, makes a class a Spring MVC controller. The Spring MVC
// parser and the recovery path both decide with it, so they agree on what a
// controller is.
func IsControllerAnnotation(name string) bool {
	return name == "RestController" || name == "Controller"
}

// parseDamaged reads a file the Kotlin grammar reported a syntax error for.
//
// The grammar rejects some valid Kotlin: an annotation class whose array
// parameter defaults to [], a stray ";" after the last enum entry, and some
// annotation argument lists that follow a property initialised to null. Most of
// these sit in the arguments of annotations whose arguments the export never
// reads, such as a validation message or range. Those argument lists are
// blanked out (positions and line numbers are kept) and the file is parsed
// again.
//
// If the grammar still reports an error, each top-level declaration is parsed
// on its own. The grammar's own recovery cannot be trusted across
// declarations: one broken member can turn its class, and every declaration
// after it, into a single ERROR node. Parsed alone, an intact declaration is
// read in full, and a damaged one is skipped and named with its line. A damaged
// declaration is skipped whole rather than member by member, because inside it
// the grammar no longer tells where one member ends; a class read in part could
// export a wrong shape, while a skipped one shows up as an unresolved type
// wherever it is referenced.
//
// A file that declares a controller still fails the run: skipping part of it
// could drop endpoints silently.
func (p *Parser) parseDamaged(path string, source []byte, firstErrorLine uint) (*ParseResult, error) {
	blanked := blankUnreadAnnotationArguments(source)
	tree := p.parser.Parse(blanked, nil)
	if tree == nil {
		return nil, fmt.Errorf("parsing %s: parser returned no tree", path)
	}
	defer tree.Close()
	root := tree.RootNode()
	if root == nil {
		return nil, fmt.Errorf("parsing %s: empty syntax tree", path)
	}

	result := &ParseResult{FilePath: path}
	if !root.HasError() {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"%s:%d: the Kotlin grammar could not parse this file as written; it was read with the arguments of "+
				"annotations the export does not use (such as validation messages) ignored", path, firstErrorLine))
		extractFile(root, blanked, path, result, false)
		return result, nil
	}

	line := firstErrorLine
	if l, ok := firstSyntaxError(root); ok {
		line = l
	}
	if declaresController(source) {
		result.Error = fmt.Errorf("%s:%d: failed to parse Kotlin source: the grammar reported a syntax error", path, line)
		return result, nil
	}

	result.Warnings = append(result.Warnings, fmt.Sprintf(
		"%s:%d: the Kotlin grammar reported a syntax error; each top-level declaration was read on its own", path, line))
	for _, decl := range topLevelDeclarations(blanked) {
		isolated := isolate(blanked, decl)
		dtree := p.parser.Parse(isolated, nil)
		if dtree == nil {
			continue
		}
		droot := dtree.RootNode()
		extractFile(droot, isolated, path, result, droot.HasError())
		dtree.Close()
	}
	return result, nil
}

// declarationDamaged reports whether a declaration holds a syntax error that
// can change what is read from it. An ERROR node that is nothing but a stray
// ";", as after the last entry of an enum, changes nothing.
func declarationDamaged(node *tree_sitter.Node, source []byte) bool {
	if !node.HasError() {
		return false
	}
	damaged := false
	var walk func(n *tree_sitter.Node)
	walk = func(n *tree_sitter.Node) {
		if damaged {
			return
		}
		if n.IsMissing() {
			damaged = true
			return
		}
		if n.IsError() {
			if !isStraySemicolon(n, source) {
				damaged = true
			}
			return
		}
		for i := uint(0); i < n.ChildCount(); i++ {
			walk(n.Child(i))
		}
	}
	walk(node)
	return damaged
}

// isStraySemicolon reports whether an ERROR node holds only ";" tokens.
func isStraySemicolon(n *tree_sitter.Node, source []byte) bool {
	if strings.TrimSpace(n.Utf8Text(source)) != ";" {
		return false
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		child := n.Child(i)
		if child.IsMissing() || child.IsError() || child.Kind() != ";" {
			return false
		}
	}
	return true
}

// declaredName names the declaration a node stands for, including an ERROR
// node that swallowed one: the identifier after its class, interface or object
// keyword.
func declaredName(node *tree_sitter.Node, source []byte) string {
	keyword := false
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "class", "interface", "object":
			keyword = true
		case "identifier":
			if keyword || node.Kind() != "ERROR" {
				return child.Utf8Text(source)
			}
		}
	}
	return ""
}

// firstSyntaxError returns the 1-based line of the first error or missing node.
func firstSyntaxError(node *tree_sitter.Node) (uint, bool) {
	if node.IsError() || node.IsMissing() {
		return node.StartPosition().Row + 1, true
	}
	if !node.HasError() {
		return 0, false
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		if line, ok := firstSyntaxError(node.Child(i)); ok {
			return line, true
		}
	}
	return 0, false
}

// span is a half-open byte range of a source.
type span struct{ start, end int }

// topLevelDeclarations splits a file into its top-level declarations, after
// the package, import and file-annotation lines. A declaration starts on a line
// that begins in the first column outside any bracket, once the declaration
// before it has reached its keyword; annotation and modifier lines above a
// declaration stay with it. Comments, strings and character literals are
// skipped while scanning.
func topLevelDeclarations(source []byte) []span {
	headerEnd := 0
	var spans []span
	start := -1
	sawKeyword := false
	// Each kind of bracket is counted on its own, so the stray ")" of a
	// damaged member cannot cancel the "{" of its class.
	var parens, braces, brackets int
	n := len(source)
	for i := 0; i < n; {
		if end, ok := skipNonCode(source, i); ok {
			i = end
			continue
		}
		depth := parens + braces + brackets
		atLineStart := i == 0 || source[i-1] == '\n'
		// Only a line that begins with a name or an annotation can start a
		// declaration; a line that begins by closing a bracket never does.
		if atLineStart && depth == 0 && (isIdentStart(source[i]) || source[i] == '@') {
			switch {
			case start < 0 && isHeaderLine(source, i):
				headerEnd = endOfLine(source, i)
				i = headerEnd
				continue
			case start < 0:
				start = i
			case sawKeyword:
				spans = append(spans, span{start, i})
				start, sawKeyword = i, false
			}
		}
		switch c := source[i]; {
		case c == '(':
			parens++
		case c == ')':
			if parens > 0 {
				parens--
			}
		case c == '{':
			braces++
		case c == '}':
			if braces > 0 {
				braces--
			}
		case c == '[':
			brackets++
		case c == ']':
			if brackets > 0 {
				brackets--
			}
		case depth == 0 && isIdentStart(c) && (i == 0 || !isIdentPart(source[i-1])):
			j := scanIdentifier(source, i)
			switch string(source[i:j]) {
			case "class", "interface", "object", "fun", "val", "var", "typealias":
				sawKeyword = true
			}
			i = j
			continue
		}
		i++
	}
	if start >= 0 {
		spans = append(spans, span{start, n})
	}
	for k := range spans {
		if spans[k].start < headerEnd {
			spans[k].start = headerEnd
		}
	}
	return spans
}

func isHeaderLine(source []byte, i int) bool {
	return hasPrefixAt(source, i, "package ") || hasPrefixAt(source, i, "import ") || hasPrefixAt(source, i, "@file:")
}

// isolate keeps the file header, the declaration, and the comments that
// belong to it: those inside its span and its own leading KDoc. Comments of
// other declarations are blanked together with their code, so leadingKDoc
// cannot attach them across the blanks to this declaration. Line breaks are
// kept so positions stay true.
func isolate(source []byte, decl span) []byte {
	out := append([]byte(nil), source...)
	keep := func(i int) bool { return i >= decl.start && i < decl.end }
	headerEnd := 0
	if spans := topLevelDeclarations(source); len(spans) > 0 {
		headerEnd = spans[0].start
	}
	// Walk by lexical units, so a string is blanked whole and a "/*" inside it
	// is not taken for a comment.
	for i := headerEnd; i < len(source); {
		end, nonCode := skipNonCode(source, i)
		if !nonCode {
			end = i + 1
		}
		comment := nonCode && isCommentStart(source, i)
		// A comment belongs to the declaration whose code follows it. Keep it
		// only when that code is kept (this declaration's span, or the leading
		// trivia that runs into it); blank the rest with their code.
		keepThis := keep(i)
		if comment && !keepThis {
			keepThis = keep(nextCode(source, end))
		}
		if !keepThis {
			for k := i; k < end && k < len(out); k++ {
				if out[k] != '\n' && out[k] != '\r' {
					out[k] = ' '
				}
			}
		}
		i = end
	}
	return out
}

// nextCode returns the first byte at or after i that starts code, skipping
// whitespace and comments. It names the declaration a comment attaches to.
func nextCode(source []byte, i int) int {
	for i < len(source) {
		switch {
		case isBlankByte(source[i]):
			i++
		case isCommentStart(source, i):
			end, _ := skipNonCode(source, i)
			i = end
		default:
			return i
		}
	}
	return len(source)
}

// blankUnreadAnnotationArguments replaces, with spaces, the argument list of
// every annotation whose arguments the collector does not read. Line breaks are
// kept so reported line numbers stay true.
func blankUnreadAnnotationArguments(source []byte) []byte {
	out := append([]byte(nil), source...)
	forEachAnnotation(source, func(name string, args int) {
		if args < 0 || argumentsRead[name] {
			return
		}
		if closing := matchParen(source, args); closing > 0 {
			for k := args + 1; k < closing; k++ {
				if out[k] != '\n' && out[k] != '\r' {
					out[k] = ' '
				}
			}
		}
	})
	return out
}

// declaresController reports whether a file annotates anything as a
// controller, by the same simple names the parser records.
func declaresController(source []byte) bool {
	found := false
	forEachAnnotation(source, func(name string, _ int) {
		if IsControllerAnnotation(name) {
			found = true
		}
	})
	return found
}

// forEachAnnotation calls visit for every annotation in code: with its simple
// name (use-site target and package qualifier dropped, as the parser records
// it) and the index of its "(" when an argument list follows, or -1. "@" signs
// inside comments, strings and character literals are not annotations.
func forEachAnnotation(source []byte, visit func(name string, args int)) {
	n := len(source)
	for i := 0; i < n; {
		if end, ok := skipNonCode(source, i); ok {
			i = end
			continue
		}
		if source[i] != '@' {
			i++
			continue
		}
		name, end := annotationName(source, i+1)
		if name == "" {
			i++
			continue
		}
		args := -1
		if end < n && source[end] == '(' {
			args = end
		}
		visit(name, args)
		if args >= 0 {
			if closing := matchParen(source, args); closing > 0 {
				// Annotations nested in the arguments are visited too.
				forEachAnnotation(source[args+1:closing], func(inner string, innerArgs int) {
					if innerArgs >= 0 {
						innerArgs += args + 1
					}
					visit(inner, innerArgs)
				})
				i = closing + 1
				continue
			}
		}
		i = end
	}
}

// annotationName reads an annotation name after "@", skipping a use-site
// target ("field:") and a package qualifier, and returns its simple name and
// the index just past it.
func annotationName(source []byte, start int) (string, int) {
	i := start
	n := len(source)
	if j := scanIdentifier(source, i); j > i && j < n && source[j] == ':' {
		i = j + 1
	}
	name := ""
	for {
		j := scanIdentifier(source, i)
		if j == i {
			return name, i
		}
		name = string(source[i:j])
		i = j
		if i < n && source[i] == '.' {
			i++
			continue
		}
		return name, i
	}
}

// matchParen returns the index of the parenthesis that closes the one at open,
// or -1 when the source ends first.
func matchParen(source []byte, open int) int {
	depth := 0
	for i := open; i < len(source); {
		if end, ok := skipNonCode(source, i); ok {
			i = end
			continue
		}
		switch source[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

// skipNonCode returns the end of the comment, string or character literal
// that starts at i, or false when i starts code.
func skipNonCode(source []byte, i int) (int, bool) {
	switch {
	case hasPrefixAt(source, i, "//"):
		return skipLineComment(source, i), true
	case hasPrefixAt(source, i, "/*"):
		return skipBlockComment(source, i), true
	case hasPrefixAt(source, i, `"""`):
		return skipRawString(source, i), true
	case source[i] == '"':
		return skipString(source, i), true
	case source[i] == '\'':
		return skipChar(source, i), true
	}
	return i, false
}

func isCommentStart(source []byte, i int) bool {
	return hasPrefixAt(source, i, "//") || hasPrefixAt(source, i, "/*")
}

func scanIdentifier(source []byte, i int) int {
	j := i
	for j < len(source) && (isIdentStart(source[j]) || (j > i && source[j] >= '0' && source[j] <= '9')) {
		j++
	}
	return j
}

func isIdentStart(c byte) bool { return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }

func isIdentPart(c byte) bool { return isIdentStart(c) || (c >= '0' && c <= '9') }

func isBlankByte(c byte) bool { return c == ' ' || c == '\t' || c == '\r' || c == '\n' }

func endOfLine(source []byte, i int) int {
	for i < len(source) && source[i] != '\n' {
		i++
	}
	return i
}

func hasPrefixAt(source []byte, i int, prefix string) bool {
	return i+len(prefix) <= len(source) && string(source[i:i+len(prefix)]) == prefix
}

func skipLineComment(source []byte, i int) int {
	return endOfLine(source, i)
}

func skipBlockComment(source []byte, i int) int {
	// Kotlin block comments nest.
	depth := 0
	for i < len(source) {
		switch {
		case hasPrefixAt(source, i, "/*"):
			depth++
			i += 2
		case hasPrefixAt(source, i, "*/"):
			depth--
			i += 2
			if depth == 0 {
				return i
			}
		default:
			i++
		}
	}
	return i
}

// skipRawString skips a """…""" string, templates included. A raw string ends
// at the last three quotes of a run, so """a"""" holds a" — the Kotlin lexer
// reads a run of three or more quotes as one closing token. Two raw strings
// cannot touch without an operator between them, so a run never spans two.
func skipRawString(source []byte, i int) int {
	i += 3
	for i < len(source) {
		switch {
		case hasPrefixAt(source, i, `"""`):
			i += 3
			for i < len(source) && source[i] == '"' {
				i++
			}
			return i
		case hasPrefixAt(source, i, "${"):
			i = skipTemplate(source, i+1)
		default:
			i++
		}
	}
	return i
}

func skipString(source []byte, i int) int {
	i++
	for i < len(source) {
		switch source[i] {
		case '\\':
			i += 2
			continue
		case '"':
			return i + 1
		case '\n':
			return i
		case '$':
			if i+1 < len(source) && source[i+1] == '{' {
				i = skipTemplate(source, i+1)
				continue
			}
		}
		i++
	}
	return i
}

// skipTemplate skips a "${...}" template expression starting at its "{". The
// expression is code: it may hold strings, character literals and comments,
// whose braces do not count.
func skipTemplate(source []byte, i int) int {
	depth := 0
	for i < len(source) {
		if end, ok := skipNonCode(source, i); ok {
			i = end
			continue
		}
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
		i++
	}
	return i
}

func skipChar(source []byte, i int) int {
	i++
	for i < len(source) {
		switch source[i] {
		case '\\':
			i += 2
			continue
		case '\'':
			return i + 1
		case '\n':
			return i
		}
		i++
	}
	return i
}

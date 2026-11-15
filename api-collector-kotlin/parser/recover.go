package parser

import (
	"fmt"
	"regexp"
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

var controllerAnnotation = regexp.MustCompile(`@(RestController|Controller)\b`)

// parseDamaged reads a file the Kotlin grammar reported a syntax error for.
//
// The grammar rejects some valid Kotlin: an annotation class whose array
// parameter defaults to [], a stray ";" after the last enum entry, and some
// annotation argument lists that follow a property initialised to null. Most of
// these sit in the arguments of annotations whose arguments the export never
// reads, such as a validation message or range. Those argument lists are
// blanked out (positions and line numbers are kept) and the file is parsed
// again. If the grammar still reports an error, the declarations it affects are
// skipped with a warning and the intact ones are kept, so one unparseable
// declaration does not abort the whole collection. A file that declares a
// controller is the exception: skipping part of it could silently drop
// endpoints, so it still fails the run.
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
	if controllerAnnotation.Match(stripComments(source)) {
		result.Error = fmt.Errorf("%s:%d: failed to parse Kotlin source: the grammar reported a syntax error", path, line)
		return result, nil
	}
	extractFile(root, blanked, path, result, true)
	result.Warnings = append(result.Warnings, fmt.Sprintf(
		"%s:%d: the Kotlin grammar reported a syntax error; declarations it affects are skipped and stay unresolved where they are referenced",
		path, line))
	return result, nil
}

// declarationDamaged reports whether a declaration holds a syntax error that
// can change what is read from it. A lone ";" the grammar could not place, as
// after the last entry of an enum, changes nothing.
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
			if strings.TrimSpace(n.Utf8Text(source)) != ";" {
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

// blankUnreadAnnotationArguments replaces, with spaces, the argument list of
// every annotation whose arguments the collector does not read. Line breaks are
// kept so reported line numbers stay true. Strings, characters and comments are
// skipped while scanning so an "@" or a parenthesis inside them is not taken
// for code.
func blankUnreadAnnotationArguments(source []byte) []byte {
	out := append([]byte(nil), source...)
	n := len(source)
	for i := 0; i < n; {
		switch {
		case hasPrefixAt(source, i, "//"):
			i = skipLineComment(source, i)
		case hasPrefixAt(source, i, "/*"):
			i = skipBlockComment(source, i)
		case hasPrefixAt(source, i, `"""`):
			i = skipRawString(source, i)
		case source[i] == '"':
			i = skipString(source, i)
		case source[i] == '\'':
			i = skipChar(source, i)
		case source[i] == '@':
			name, end := annotationName(source, i+1)
			if name == "" {
				i++
				continue
			}
			if end < n && source[end] == '(' && !argumentsRead[name] {
				if closing := matchParen(source, end); closing > 0 {
					for k := end + 1; k < closing; k++ {
						if out[k] != '\n' && out[k] != '\r' {
							out[k] = ' '
						}
					}
					i = closing + 1
					continue
				}
			}
			i = end
		default:
			i++
		}
	}
	return out
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

func scanIdentifier(source []byte, i int) int {
	j := i
	for j < len(source) {
		c := source[j]
		if c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (j > i && c >= '0' && c <= '9') {
			j++
			continue
		}
		break
	}
	return j
}

// matchParen returns the index of the parenthesis that closes the one at open,
// or -1 when the source ends first.
func matchParen(source []byte, open int) int {
	depth := 0
	for i := open; i < len(source); {
		switch {
		case hasPrefixAt(source, i, "//"):
			i = skipLineComment(source, i)
		case hasPrefixAt(source, i, "/*"):
			i = skipBlockComment(source, i)
		case hasPrefixAt(source, i, `"""`):
			i = skipRawString(source, i)
		case source[i] == '"':
			i = skipString(source, i)
		case source[i] == '\'':
			i = skipChar(source, i)
		default:
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
	}
	return -1
}

// stripComments returns source with comments blanked, so a controller
// annotation mentioned in a comment is not taken for a declaration.
func stripComments(source []byte) []byte {
	out := append([]byte(nil), source...)
	for i := 0; i < len(source); {
		switch {
		case hasPrefixAt(source, i, "//"):
			end := skipLineComment(source, i)
			blank(out, i, end)
			i = end
		case hasPrefixAt(source, i, "/*"):
			end := skipBlockComment(source, i)
			blank(out, i, end)
			i = end
		case hasPrefixAt(source, i, `"""`):
			i = skipRawString(source, i)
		case source[i] == '"':
			i = skipString(source, i)
		default:
			i++
		}
	}
	return out
}

func blank(out []byte, from, to int) {
	for k := from; k < to && k < len(out); k++ {
		if out[k] != '\n' && out[k] != '\r' {
			out[k] = ' '
		}
	}
}

func hasPrefixAt(source []byte, i int, prefix string) bool {
	return i+len(prefix) <= len(source) && string(source[i:i+len(prefix)]) == prefix
}

func skipLineComment(source []byte, i int) int {
	for i < len(source) && source[i] != '\n' {
		i++
	}
	return i
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

func skipRawString(source []byte, i int) int {
	i += 3
	for i < len(source) {
		if hasPrefixAt(source, i, `"""`) {
			// A raw string may end with extra quotes: """a""""
			i += 3
			for i < len(source) && source[i] == '"' {
				i++
			}
			return i
		}
		i++
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

// skipTemplate skips a "${...}" template expression starting at its "{",
// which may itself hold strings.
func skipTemplate(source []byte, i int) int {
	depth := 0
	for i < len(source) {
		switch {
		case hasPrefixAt(source, i, `"""`):
			i = skipRawString(source, i)
			continue
		case source[i] == '"':
			i = skipString(source, i)
			continue
		case source[i] == '{':
			depth++
		case source[i] == '}':
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

package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	kotlin "github.com/tree-sitter-grammars/tree-sitter-kotlin/bindings/go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// Parser wraps the tree-sitter Kotlin parser.
type Parser struct {
	parser *tree_sitter.Parser
}

// NewParser creates a Kotlin parser.
func NewParser() (*Parser, error) {
	p := tree_sitter.NewParser()
	if err := p.SetLanguage(tree_sitter.NewLanguage(kotlin.Language())); err != nil {
		p.Close()
		return nil, fmt.Errorf("loading Kotlin grammar: %w", err)
	}
	return &Parser{parser: p}, nil
}

func (p *Parser) Close() {
	if p != nil && p.parser != nil {
		p.parser.Close()
	}
}

// ParseFile parses one .kt file.
func (p *Parser) ParseFile(path string) (*ParseResult, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return p.ParseSource(path, source)
}

// ParseSource parses Kotlin source read from elsewhere, such as a class of a
// dependency's sources JAR. path names the source in messages.
func (p *Parser) ParseSource(path string, source []byte) (*ParseResult, error) {
	tree := p.parser.Parse(source, nil)
	if tree == nil {
		return nil, fmt.Errorf("parsing %s: parser returned no tree", path)
	}
	defer tree.Close()

	root := tree.RootNode()
	if root == nil {
		return nil, fmt.Errorf("parsing %s: empty syntax tree", path)
	}
	if root.HasError() {
		line, _ := firstSyntaxError(root)
		return p.parseDamaged(path, source, line)
	}

	result := &ParseResult{FilePath: path}
	extractFile(root, source, path, result, false)
	return result, nil
}

// ParseDirectory parses every .kt file under dir. Build output and dependency
// directories are skipped so a Gradle project does not export generated code.
func (p *Parser) ParseDirectory(dir string) ([]ParseResult, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			if path != dir && skipDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), ".kt") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking %s: %w", dir, err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, &NoKotlinSourcesError{Dir: dir}
	}

	results := make([]ParseResult, 0, len(files))
	for _, file := range files {
		result, parseErr := p.ParseFile(file)
		if parseErr != nil {
			return nil, parseErr
		}
		results = append(results, *result)
	}
	return results, nil
}

func skipDir(name string) bool {
	switch name {
	case ".git", ".gradle", ".idea", ".kotlin", "build", "out", "target", "node_modules":
		return true
	}
	return false
}

func isTypeNode(kind string) bool {
	switch kind {
	case "user_type", "nullable_type", "function_type", "parenthesized_type", "dynamic_type",
		"intersection_type", "type_reference", "type_projection", "star_projection":
		return true
	}
	return false
}

func firstChildOfKind(node *tree_sitter.Node, kinds ...string) *tree_sitter.Node {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		for _, kind := range kinds {
			if child.Kind() == kind {
				return child
			}
		}
	}
	return nil
}

func directIndexOf(node *tree_sitter.Node, kind string) int {
	for i := uint(0); i < node.ChildCount(); i++ {
		if node.Child(i).Kind() == kind {
			return int(i)
		}
	}
	return -1
}

func lastSegment(name string) string {
	name = strings.TrimSpace(name)
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

// positionText returns the source text that starts at the given byte offset.
func positionText(source []byte, start uint, end uint) string {
	if start >= uint(len(source)) {
		return ""
	}
	if end == 0 || end > uint(len(source)) {
		end = uint(len(source))
	}
	return string(source[start:end])
}

// cleanTypeText removes nullability markers, variance and whitespace from a
// type spelling. Requiredness never comes from "?" — it comes from annotations.
func cleanTypeText(raw string) string {
	raw = strings.ReplaceAll(raw, "\r", " ")
	raw = strings.ReplaceAll(raw, "\n", " ")
	raw = strings.ReplaceAll(raw, "?", "")
	raw = strings.ReplaceAll(raw, "out ", "")
	raw = strings.ReplaceAll(raw, "in ", "")
	raw = strings.Join(strings.Fields(raw), "")
	return strings.TrimSpace(raw)
}

// extractTypeNode returns the cleaned spelling of a type node.
func extractTypeNode(node *tree_sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	switch node.Kind() {
	case "nullable_type":
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if isTypeNode(child.Kind()) {
				return extractTypeNode(child, source)
			}
		}
	case "parenthesized_type":
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if isTypeNode(child.Kind()) {
				return extractTypeNode(child, source)
			}
		}
	case "type_projection":
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() == "star_projection" {
				return "*"
			}
			if isTypeNode(child.Kind()) {
				return extractTypeNode(child, source)
			}
		}
	case "star_projection":
		return "*"
	}
	return cleanTypeText(typeSource(node, source))
}

// typeSource returns the source text of a type node without the annotations
// written inside it: `List<@NotBlank(message = X) String>` reads as
// `List<String>`.
//
// An annotation in a type argument is a type annotation. It says something of
// the elements, not of the declaration that holds the type, so it is no part of
// the type's name; kept in the spelling, it made `@NotBlank(message = X) String`
// a type of its own that no declaration could be found for. The annotations of
// the declaration itself are read from its modifiers, and are left alone.
func typeSource(node *tree_sitter.Node, source []byte) string {
	var cuts [][2]uint
	collectTypeModifiers(node, &cuts)
	if len(cuts) == 0 {
		return node.Utf8Text(source)
	}

	start, end := node.StartByte(), node.EndByte()
	var out []byte
	pos := start
	for _, cut := range cuts {
		out = append(out, source[pos:cut[0]]...)
		// A space stays where the modifiers were, so that the tokens on either
		// side do not run together.
		out = append(out, ' ')
		pos = cut[1]
	}
	out = append(out, source[pos:end]...)
	return string(out)
}

// collectTypeModifiers gathers the byte ranges of the type modifiers under a
// type node, in source order. A type_modifiers node holds the annotations of
// one type; its own arguments sit inside the range that is cut, so the search
// does not go into it.
func collectTypeModifiers(node *tree_sitter.Node, cuts *[][2]uint) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "type_modifiers" {
			*cuts = append(*cuts, [2]uint{child.StartByte(), child.EndByte()})
			continue
		}
		collectTypeModifiers(child, cuts)
	}
}

// typeAfterColon finds the declared type after the ":" token in a declaration.
func typeAfterColon(node *tree_sitter.Node, source []byte) string {
	sawColon := false
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == ":" {
			sawColon = true
			continue
		}
		if sawColon && isTypeNode(child.Kind()) {
			return extractTypeNode(child, source)
		}
	}
	return ""
}

// nullableTypeAfterColon reports whether the declared type after the ":"
// token is nullable as a whole: `String?` and `List<String>?` are, while
// `List<String?>` is not.
func nullableTypeAfterColon(node *tree_sitter.Node, source []byte) bool {
	sawColon := false
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == ":" {
			sawColon = true
			continue
		}
		if sawColon && isTypeNode(child.Kind()) {
			return strings.HasSuffix(strings.TrimSpace(child.Utf8Text(source)), "?")
		}
	}
	return false
}

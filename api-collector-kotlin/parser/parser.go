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
		return &ParseResult{FilePath: path, Error: fmt.Errorf("%s: failed to parse Kotlin source: the grammar reported a syntax error", path)}, nil
	}

	result := &ParseResult{FilePath: path}
	extractFile(root, source, path, result)
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
	return cleanTypeText(node.Utf8Text(source))
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

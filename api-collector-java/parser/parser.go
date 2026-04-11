// Package parser provides Tree-sitter based Java/Kotlin source parsing.
package parser

import (
	"fmt"
	"os"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	java "github.com/tree-sitter/tree-sitter-java/bindings/go"
)

const (
	// maxWalkDepth prevents stack overflow in deeply nested AST structures
	maxWalkDepth = 1000
)

// Parser wraps tree-sitter parser for Java source files.
type Parser struct {
	parser *tree_sitter.Parser
}

// NewParser creates a new Java parser.
func NewParser() (*Parser, error) {
	parser := tree_sitter.NewParser()
	language := tree_sitter.NewLanguage(java.Language())

	if err := parser.SetLanguage(language); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	return &Parser{parser: parser}, nil
}

// ParseFile parses a Java source file and returns the AST.
func (p *Parser) ParseFile(path string) (*tree_sitter.Tree, []byte, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	tree := p.parser.Parse(source, nil)
	if tree == nil {
		return nil, nil, fmt.Errorf("failed to parse file")
	}

	return tree, source, nil
}

// ExtractClasses extracts all classes with their annotations and methods.
func (p *Parser) ExtractClasses(tree *tree_sitter.Tree, source []byte) ([]Class, error) {
	var classes []Class
	var packageName string

	rootNode := tree.RootNode()
	cursor := rootNode.Walk()
	defer cursor.Close()

	// Extract package name first
	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child.Kind() == "package_declaration" {
			packageName = extractPackageName(child, source)
			break
		}
	}

	// Walk the tree to find class declarations
	p.walkTree(cursor, source, func(node *tree_sitter.Node) {
		if node.Kind() == "class_declaration" {
			class := extractClass(node, source)
			class.Package = packageName
			classes = append(classes, class)
		}
	})

	return classes, nil
}

// walkTree walks the AST and calls the callback for each node.
func (p *Parser) walkTree(cursor *tree_sitter.TreeCursor, source []byte, callback func(*tree_sitter.Node)) {
	p.walkTreeWithDepth(cursor, source, callback, 0)
}

// walkTreeWithDepth walks the AST with depth limit to prevent stack overflow.
func (p *Parser) walkTreeWithDepth(cursor *tree_sitter.TreeCursor, source []byte, callback func(*tree_sitter.Node), depth int) {
	if depth > maxWalkDepth {
		return
	}

	node := cursor.Node()
	callback(node)

	if cursor.GotoFirstChild() {
		p.walkTreeWithDepth(cursor, source, callback, depth+1)
		cursor.GotoParent()
	}

	if cursor.GotoNextSibling() {
		p.walkTreeWithDepth(cursor, source, callback, depth)
	}
}

// Close releases parser resources.
func (p *Parser) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
}

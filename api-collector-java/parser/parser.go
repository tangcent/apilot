// Package parser provides Tree-sitter based Java/Kotlin source parsing.
package parser

import (
	"fmt"
	"os"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	java "github.com/tree-sitter/tree-sitter-java/bindings/go"
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

	rootNode := tree.RootNode()
	cursor := rootNode.Walk()
	defer cursor.Close()

	// Walk the tree to find class declarations
	p.walkTree(cursor, source, func(node *tree_sitter.Node) {
		if node.Kind() == "class_declaration" {
			class := p.extractClass(node, source)
			classes = append(classes, class)
		}
	})

	return classes, nil
}

// extractClass extracts class information including annotations and methods.
func (p *Parser) extractClass(node *tree_sitter.Node, source []byte) Class {
	class := Class{}

	// Extract class name
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" {
			class.Name = child.Utf8Text(source)
			break
		}
	}

	// Extract class annotations
	class.Annotations = p.extractAnnotations(node, source)

	// Extract methods
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "class_body" {
			class.Methods = p.extractMethods(child, source)
			break
		}
	}

	return class
}

// extractAnnotations extracts annotations from a node.
func (p *Parser) extractAnnotations(node *tree_sitter.Node, source []byte) []Annotation {
	var annotations []Annotation

	// Look for modifiers node which contains annotations
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "modifiers" {
			for j := uint(0); j < child.ChildCount(); j++ {
				modChild := child.Child(j)
				if modChild.Kind() == "marker_annotation" || modChild.Kind() == "annotation" {
					ann := p.extractAnnotation(modChild, source)
					annotations = append(annotations, ann)
				}
			}
		}
	}

	return annotations
}

// extractAnnotation extracts a single annotation.
func (p *Parser) extractAnnotation(node *tree_sitter.Node, source []byte) Annotation {
	ann := Annotation{
		Params: make(map[string]string),
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "identifier", "scoped_identifier":
			ann.Name = child.Utf8Text(source)
		case "annotation_argument_list":
			// Extract annotation parameters
			p.extractAnnotationParams(child, source, ann.Params)
		}
	}

	return ann
}

// extractAnnotationParams extracts annotation parameters.
func (p *Parser) extractAnnotationParams(node *tree_sitter.Node, source []byte, params map[string]string) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "element_value_pair" {
			// Named parameter: @GetMapping(value = "/users")
			var key, value string
			for j := uint(0); j < child.ChildCount(); j++ {
				subChild := child.Child(j)
				if subChild.Kind() == "identifier" {
					key = subChild.Utf8Text(source)
				} else if subChild.Kind() == "string_literal" {
					value = p.extractStringLiteral(subChild, source)
				}
			}
			if key != "" {
				params[key] = value
			}
		} else if child.Kind() == "string_literal" {
			// Single value parameter: @GetMapping("/users")
			params["value"] = p.extractStringLiteral(child, source)
		}
	}
}

// extractStringLiteral extracts string content without quotes.
func (p *Parser) extractStringLiteral(node *tree_sitter.Node, source []byte) string {
	text := node.Utf8Text(source)
	// Remove surrounding quotes
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return text[1 : len(text)-1]
	}
	return text
}

// extractMethods extracts all methods from a class body.
func (p *Parser) extractMethods(classBody *tree_sitter.Node, source []byte) []Method {
	var methods []Method

	for i := uint(0); i < classBody.ChildCount(); i++ {
		child := classBody.Child(i)
		if child.Kind() == "method_declaration" {
			method := p.extractMethod(child, source)
			methods = append(methods, method)
		}
	}

	return methods
}

// extractMethod extracts method information.
func (p *Parser) extractMethod(node *tree_sitter.Node, source []byte) Method {
	method := Method{}

	// Extract annotations
	method.Annotations = p.extractAnnotations(node, source)

	// Extract method name, return type, and parameters
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "identifier":
			method.Name = child.Utf8Text(source)
		case "type_identifier", "generic_type":
			method.ReturnType = child.Utf8Text(source)
		case "formal_parameters":
			method.Parameters = p.extractParameters(child, source)
		}
	}

	return method
}

// extractParameters extracts method parameters.
func (p *Parser) extractParameters(node *tree_sitter.Node, source []byte) []Parameter {
	var params []Parameter

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "formal_parameter" {
			param := p.extractParameter(child, source)
			params = append(params, param)
		}
	}

	return params
}

// extractParameter extracts a single parameter.
func (p *Parser) extractParameter(node *tree_sitter.Node, source []byte) Parameter {
	param := Parameter{}

	// Extract annotations
	param.Annotations = p.extractAnnotations(node, source)

	// Extract type and name
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "type_identifier", "generic_type", "integral_type":
			param.Type = child.Utf8Text(source)
		case "identifier":
			param.Name = child.Utf8Text(source)
		}
	}

	return param
}

// walkTree walks the AST and calls the callback for each node.
func (p *Parser) walkTree(cursor *tree_sitter.TreeCursor, source []byte, callback func(*tree_sitter.Node)) {
	node := cursor.Node()
	callback(node)

	if cursor.GotoFirstChild() {
		p.walkTree(cursor, source, callback)
		cursor.GotoParent()
	}

	if cursor.GotoNextSibling() {
		p.walkTree(cursor, source, callback)
	}
}

// Close releases parser resources.
func (p *Parser) Close() {
	if p.parser != nil {
		p.parser.Close()
	}
}

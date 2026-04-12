package parser

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// extractPackageName extracts package name from package_declaration node.
func extractPackageName(node *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "scoped_identifier" || child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

// extractClass extracts class information including annotations and methods.
func extractClass(node *tree_sitter.Node, source []byte) Class {
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
	class.Annotations = extractAnnotations(node, source)

	// Extract methods
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "class_body" {
			class.Methods = extractMethods(child, source)
			break
		}
	}

	return class
}

// extractInterface extracts interface information including annotations and methods.
func extractInterface(node *tree_sitter.Node, source []byte) Class {
	class := Class{IsInterface: true}

	// Extract interface name
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" {
			class.Name = child.Utf8Text(source)
			break
		}
	}

	// Extract interface annotations
	class.Annotations = extractAnnotations(node, source)

	// Extract methods from interface body
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "interface_body" {
			class.Methods = extractMethods(child, source)
			break
		}
	}

	return class
}

// extractAnnotations extracts annotations from a node.
func extractAnnotations(node *tree_sitter.Node, source []byte) []Annotation {
	var annotations []Annotation

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "modifiers" {
			for j := uint(0); j < child.ChildCount(); j++ {
				modChild := child.Child(j)
				if modChild.Kind() == "marker_annotation" || modChild.Kind() == "annotation" {
					ann := extractAnnotation(modChild, source)
					annotations = append(annotations, ann)
				}
			}
		}
	}

	return annotations
}

// extractAnnotation extracts a single annotation.
func extractAnnotation(node *tree_sitter.Node, source []byte) Annotation {
	ann := Annotation{
		Params: make(map[string]string),
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "identifier", "scoped_identifier":
			ann.Name = child.Utf8Text(source)
		case "annotation_argument_list":
			extractAnnotationParams(child, source, ann.Params)
		}
	}

	return ann
}

// extractAnnotationParams extracts annotation parameters.
func extractAnnotationParams(node *tree_sitter.Node, source []byte, params map[string]string) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "element_value_pair" {
			var key, value string
			for j := uint(0); j < child.ChildCount(); j++ {
				subChild := child.Child(j)
				if subChild.Kind() == "identifier" {
					key = subChild.Utf8Text(source)
				} else if subChild.Kind() == "string_literal" {
					value = extractStringLiteral(subChild, source)
				}
			}
			if key != "" {
				params[key] = value
			}
		} else if child.Kind() == "string_literal" {
			params["value"] = extractStringLiteral(child, source)
		}
	}
}

// extractStringLiteral extracts string content without quotes.
func extractStringLiteral(node *tree_sitter.Node, source []byte) string {
	text := node.Utf8Text(source)
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return text[1 : len(text)-1]
	}
	return text
}

// extractMethods extracts all methods from a class body.
func extractMethods(classBody *tree_sitter.Node, source []byte) []Method {
	var methods []Method

	for i := uint(0); i < classBody.ChildCount(); i++ {
		child := classBody.Child(i)
		if child.Kind() == "method_declaration" {
			method := extractMethod(child, source)
			methods = append(methods, method)
		}
	}

	return methods
}

// extractMethod extracts method information.
func extractMethod(node *tree_sitter.Node, source []byte) Method {
	method := Method{}

	method.Annotations = extractAnnotations(node, source)

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "identifier":
			method.Name = child.Utf8Text(source)
		case "type_identifier", "generic_type":
			method.ReturnType = child.Utf8Text(source)
		case "formal_parameters":
			method.Parameters = extractParameters(child, source)
		}
	}

	return method
}

// extractParameters extracts method parameters.
func extractParameters(node *tree_sitter.Node, source []byte) []Parameter {
	var params []Parameter

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "formal_parameter" {
			param := extractParameter(child, source)
			params = append(params, param)
		}
	}

	return params
}

// extractParameter extracts a single parameter.
func extractParameter(node *tree_sitter.Node, source []byte) Parameter {
	param := Parameter{}

	param.Annotations = extractAnnotations(node, source)

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

// walkTreeWithDepth walks the AST with depth limit to prevent stack overflow.
func walkTreeWithDepth(cursor *tree_sitter.TreeCursor, source []byte, callback func(*tree_sitter.Node), depth int) {
	if depth > maxWalkDepth {
		return
	}

	node := cursor.Node()
	callback(node)

	if cursor.GotoFirstChild() {
		walkTreeWithDepth(cursor, source, callback, depth+1)
		cursor.GotoParent()
	}

	if cursor.GotoNextSibling() {
		walkTreeWithDepth(cursor, source, callback, depth)
	}
}

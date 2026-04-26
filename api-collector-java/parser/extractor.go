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

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" {
			class.Name = child.Utf8Text(source)
			break
		}
	}

	class.Annotations = extractAnnotations(node, source)
	class.SuperClass, class.SuperClassTypeArgs = extractSuperClass(node, source)
	class.TypeParameters = extractTypeParameters(node, source)
	class.Interfaces = extractInterfaces(node, source)

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "class_body" {
			class.Methods = extractMethods(child, source)
			class.Fields = extractFields(child, source)
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
		case "type_identifier", "generic_type", "integral_type", "floating_point_type", "boolean_type", "void_type":
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
		case "type_identifier", "generic_type", "integral_type", "floating_point_type", "boolean_type":
			param.Type = child.Utf8Text(source)
		case "identifier":
			param.Name = child.Utf8Text(source)
		}
	}

	return param
}

// extractFields extracts all field declarations from a class body.
func extractFields(classBody *tree_sitter.Node, source []byte) []Field {
	var fields []Field

	for i := uint(0); i < classBody.ChildCount(); i++ {
		child := classBody.Child(i)
		if child.Kind() == "field_declaration" {
			if field := extractField(child, source); field != nil {
				fields = append(fields, *field)
			}
		}
	}

	return fields
}

// extractField extracts a single field declaration.
func extractField(node *tree_sitter.Node, source []byte) *Field {
	field := &Field{}

	field.Annotations = extractAnnotations(node, source)

	var fieldType string
	var fieldName string

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "modifiers":
			for j := uint(0); j < child.ChildCount(); j++ {
				modChild := child.Child(j)
				switch modChild.Kind() {
				case "static":
					field.IsStatic = true
				case "final":
					field.IsFinal = true
				}
			}
		case "type_identifier", "generic_type", "integral_type", "floating_point_type", "boolean_type":
			fieldType = child.Utf8Text(source)
		case "variable_declarator":
			for j := uint(0); j < child.ChildCount(); j++ {
				vdChild := child.Child(j)
				if vdChild.Kind() == "identifier" {
					fieldName = vdChild.Utf8Text(source)
					break
				}
			}
		}
	}

	if fieldType == "" || fieldName == "" {
		return nil
	}

	field.Type = fieldType
	field.Name = fieldName

	return field
}

// extractSuperClass extracts superclass name and type arguments from a class declaration.
func extractSuperClass(node *tree_sitter.Node, source []byte) (string, []string) {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "superclass" {
			for j := uint(0); j < child.ChildCount(); j++ {
				scChild := child.Child(j)
				switch scChild.Kind() {
				case "type_identifier":
					return scChild.Utf8Text(source), nil
				case "generic_type":
					return extractGenericBaseAndArgs(scChild, source)
				}
			}
		}
	}
	return "", nil
}

// extractInterfaces extracts interface names from a class declaration's implements clause.
func extractInterfaces(node *tree_sitter.Node, source []byte) []string {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "super_interfaces" {
			var ifaces []string
			for j := uint(0); j < child.ChildCount(); j++ {
				ifaceChild := child.Child(j)
				switch ifaceChild.Kind() {
				case "type_identifier":
					ifaces = append(ifaces, ifaceChild.Utf8Text(source))
				case "generic_type":
					baseName, _ := extractGenericBaseAndArgs(ifaceChild, source)
					ifaces = append(ifaces, baseName)
				case "type_list":
					for k := uint(0); k < ifaceChild.ChildCount(); k++ {
						listChild := ifaceChild.Child(k)
						switch listChild.Kind() {
						case "type_identifier":
							ifaces = append(ifaces, listChild.Utf8Text(source))
						case "generic_type":
							baseName, _ := extractGenericBaseAndArgs(listChild, source)
							ifaces = append(ifaces, baseName)
						}
					}
				}
			}
			return ifaces
		}
	}
	return nil
}

// extractGenericBaseAndArgs extracts the base name and type arguments from a generic_type node.
func extractGenericBaseAndArgs(node *tree_sitter.Node, source []byte) (string, []string) {
	var baseName string
	var typeArgs []string

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "type_identifier":
			baseName = child.Utf8Text(source)
		case "type_arguments":
			typeArgs = extractTypeArguments(child, source)
		}
	}

	return baseName, typeArgs
}

// extractTypeArguments extracts type argument strings from a type_arguments node.
func extractTypeArguments(node *tree_sitter.Node, source []byte) []string {
	var args []string

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "type_identifier":
			args = append(args, child.Utf8Text(source))
		case "generic_type":
			args = append(args, child.Utf8Text(source))
		}
	}

	return args
}

// extractTypeParameters extracts type parameter names from a class declaration.
func extractTypeParameters(node *tree_sitter.Node, source []byte) []string {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "type_parameters" {
			var params []string
			for j := uint(0); j < child.ChildCount(); j++ {
				tpChild := child.Child(j)
				if tpChild.Kind() == "type_parameter" {
					for k := uint(0); k < tpChild.ChildCount(); k++ {
						identChild := tpChild.Child(k)
						if identChild.Kind() == "identifier" || identChild.Kind() == "type_identifier" {
							params = append(params, identChild.Utf8Text(source))
							break
						}
					}
				}
			}
			return params
		}
	}
	return nil
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

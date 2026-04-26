package flask

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type MarshmallowModel struct {
	Name          string
	Fields        []MarshmallowField
	EmbeddedTypes []string
}

type MarshmallowField struct {
	Name      string
	FieldType string
	Required  bool
	Many      bool
	Nested    string
}

var marshmallowFieldTypeMap = map[string]string{
	"String":  "string",
	"Str":     "string",
	"Integer": "int",
	"Int":     "int",
	"Float":   "float",
	"Boolean": "boolean",
	"Bool":    "boolean",
	"DateTime": "string",
	"Date":    "string",
	"Time":    "string",
	"Email":   "string",
	"URL":     "string",
	"Url":     "string",
	"UUID":    "string",
	"IP":      "string",
	"IPv4":    "string",
	"IPv6":    "string",
	"MAC":     "string",
	"Decimal": "float",
	"Raw":     "string",
	"Dict":    "map",
	"List":    "array",
	"Tuple":   "array",
	"Nested":  "object",
	"Method":  "string",
	"Function": "string",
	"Constant": "string",
	"File":    "string",
}

func extractMarshmallowSchemas(rootNode *tree_sitter.Node, source []byte) map[string]MarshmallowModel {
	allClasses := make(map[string]*marshmallowClassInfo)

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child.Kind() == "class_definition" {
			info := extractMarshmallowClassInfo(child, source)
			if info != nil {
				allClasses[info.name] = info
			}
		}
	}

	schemaSet := findMarshmallowSchemas(allClasses)

	models := make(map[string]MarshmallowModel)
	for name, info := range allClasses {
		if !schemaSet[name] {
			continue
		}
		md := MarshmallowModel{
			Name:   name,
			Fields: info.fields,
		}
		for _, parent := range info.parents {
			if !isMarshmallowBaseClass(parent) {
				md.EmbeddedTypes = append(md.EmbeddedTypes, parent)
			}
		}
		models[name] = md
	}

	return models
}

type marshmallowClassInfo struct {
	name    string
	parents []string
	fields  []MarshmallowField
}

func extractMarshmallowClassInfo(node *tree_sitter.Node, source []byte) *marshmallowClassInfo {
	var name string
	var argList *tree_sitter.Node
	var body *tree_sitter.Node

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "identifier":
			name = child.Utf8Text(source)
		case "argument_list":
			argList = child
		case "block":
			body = child
		}
	}

	if name == "" {
		return nil
	}

	info := &marshmallowClassInfo{name: name}

	if argList != nil {
		info.parents = extractMarshmallowParentClasses(argList, source)
	}

	if body != nil {
		info.fields = extractMarshmallowFields(body, source)
	}

	return info
}

func extractMarshmallowParentClasses(argList *tree_sitter.Node, source []byte) []string {
	var parents []string
	for i := uint(0); i < argList.ChildCount(); i++ {
		child := argList.Child(i)
		if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
			continue
		}
		text := strings.TrimSpace(child.Utf8Text(source))
		if text != "" {
			parents = append(parents, text)
		}
	}
	return parents
}

func findMarshmallowSchemas(allClasses map[string]*marshmallowClassInfo) map[string]bool {
	schemaSet := make(map[string]bool)

	changed := true
	for changed {
		changed = false
		for name, info := range allClasses {
			if schemaSet[name] {
				continue
			}
			for _, parent := range info.parents {
				if isMarshmallowBaseClass(parent) {
					schemaSet[name] = true
					changed = true
					break
				}
				if schemaSet[parent] {
					schemaSet[name] = true
					changed = true
					break
				}
			}
		}
	}

	return schemaSet
}

func isMarshmallowBaseClass(parent string) bool {
	switch parent {
	case "Schema", "ModelSchema", "TableSchema",
		"SQLAlchemyAutoSchema", "SQLAlchemySchema",
		"HyperlinkRelatedField":
		return true
	}
	if strings.HasSuffix(parent, ".Schema") ||
		strings.HasSuffix(parent, ".ModelSchema") ||
		strings.HasSuffix(parent, ".SQLAlchemyAutoSchema") ||
		strings.HasSuffix(parent, ".SQLAlchemySchema") {
		return true
	}
	return false
}

func extractMarshmallowFields(body *tree_sitter.Node, source []byte) []MarshmallowField {
	var fields []MarshmallowField

	for i := uint(0); i < body.ChildCount(); i++ {
		child := body.Child(i)
		if child.Kind() != "expression_statement" {
			continue
		}

		for j := uint(0); j < child.ChildCount(); j++ {
			subChild := child.Child(j)
			if subChild.Kind() == "assignment" {
				f := extractMarshmallowFieldFromAssignment(subChild, source)
				if f != nil {
					fields = append(fields, *f)
				}
			}
		}
	}

	return fields
}

func extractMarshmallowFieldFromAssignment(assignNode *tree_sitter.Node, source []byte) *MarshmallowField {
	var name string
	var rightSide *tree_sitter.Node

	for i := uint(0); i < assignNode.ChildCount(); i++ {
		child := assignNode.Child(i)
		switch child.Kind() {
		case "identifier":
			if name == "" {
				name = child.Utf8Text(source)
			}
		case "=":
			// skip
		default:
			if name != "" && rightSide == nil {
				rightSide = child
			}
		}
	}

	if name == "" || rightSide == nil {
		return nil
	}

	if rightSide.Kind() == "call" {
		return extractMarshmallowFieldFromCall(name, rightSide, source)
	}

	return nil
}

func extractMarshmallowFieldFromCall(name string, callNode *tree_sitter.Node, source []byte) *MarshmallowField {
	var fieldType string
	var required bool = true
	var many bool
	var nested string

	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "attribute" {
			fieldType = extractMarshmallowFieldTypeName(child, source)
		} else if child.Kind() == "identifier" {
			fieldType = child.Utf8Text(source)
		} else if child.Kind() == "argument_list" {
			required, many, nested = extractMarshmallowFieldArgs(child, source, fieldType)
		}
	}

	if fieldType == "" {
		return nil
	}

	if _, ok := marshmallowFieldTypeMap[fieldType]; !ok && nested == "" {
		nested = fieldType
		fieldType = "Nested"
	}

	return &MarshmallowField{
		Name:      name,
		FieldType: fieldType,
		Required:  required,
		Many:      many,
		Nested:    nested,
	}
}

func extractMarshmallowFieldTypeName(attrNode *tree_sitter.Node, source []byte) string {
	var obj string
	var attr string

	for i := uint(0); i < attrNode.ChildCount(); i++ {
		child := attrNode.Child(i)
		if child.Kind() == "." {
			continue
		}
		if child.Kind() == "identifier" {
			if obj == "" {
				obj = child.Utf8Text(source)
			} else {
				attr = child.Utf8Text(source)
			}
		}
		if child.Kind() == "attribute" {
			innerAttr := extractMarshmallowFieldTypeName(child, source)
			if innerAttr != "" {
				attr = innerAttr
			}
		}
	}

	if attr != "" {
		return attr
	}
	return obj
}

func extractMarshmallowFieldArgs(argList *tree_sitter.Node, source []byte, fieldType string) (required bool, many bool, nested string) {
	required = true

	if fieldType == "Nested" {
		for i := uint(0); i < argList.ChildCount(); i++ {
			child := argList.Child(i)
			if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
				continue
			}
			if child.Kind() == "identifier" && nested == "" {
				nested = child.Utf8Text(source)
			}
			if child.Kind() == "keyword_argument" {
				kwName, kwValue := extractKeywordArg(child, source)
				switch kwName {
				case "required":
					required = kwValue != "False"
				case "many":
					many = kwValue == "True"
				case "nested":
					nested = kwValue
				}
			}
		}
		return required, many, nested
	}

	for i := uint(0); i < argList.ChildCount(); i++ {
		child := argList.Child(i)
		if child.Kind() == "keyword_argument" {
			kwName, kwValue := extractKeywordArg(child, source)
			switch kwName {
			case "required":
				required = kwValue != "False"
			case "many":
				many = kwValue == "True"
			case "nested":
				nested = kwValue
			}
		}
	}

	return required, many, nested
}

func extractKeywordArg(kwArgNode *tree_sitter.Node, source []byte) (name string, value string) {
	for i := uint(0); i < kwArgNode.ChildCount(); i++ {
		child := kwArgNode.Child(i)
		if child.Kind() == "identifier" && name == "" {
			name = child.Utf8Text(source)
		} else if child.Kind() != "=" && name != "" && value == "" {
			value = child.Utf8Text(source)
		}
	}
	return name, value
}

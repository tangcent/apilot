package fastify

import (
	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	collector "github.com/tangcent/apilot/api-collector"
	docmeta "github.com/tangcent/apilot/api-docmeta"
	model "github.com/tangcent/apilot/api-model"
)

// extractSchemaFieldDoc reads documentation keywords (description, default,
// enum, example) from a JSON Schema property object.
func extractSchemaFieldDoc(objNode *tree_sitter.Node, source []byte) docmeta.Documentation {
	var d docmeta.Documentation
	for i := uint(0); i < objNode.ChildCount(); i++ {
		child := objNode.Child(i)
		if child.Kind() != "pair" {
			continue
		}
		switch extractPairKey(child, source) {
		case "description":
			d.Comment = extractPairStringValue(child, source)
		case "default":
			d.DefaultValue = extractPairValueText(child, source)
		case "example":
			d.Demo = extractPairValueText(child, source)
		case "enum":
			d.Options = docmeta.OptionsFromValues(extractSchemaStringArray(child, source))
		}
	}
	return d
}

// extractPairValueText returns the value of a pair as literal text, with
// string quotes stripped.
func extractPairValueText(pairNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < pairNode.ChildCount(); i++ {
		child := pairNode.Child(i)
		switch child.Kind() {
		case "string":
			return unquoteJSString(child.Utf8Text(source))
		case "number", "true", "false", "null", "identifier":
			return child.Utf8Text(source)
		}
	}
	return ""
}

// extractSchemaStringArray collects the string elements of an array-valued
// pair, e.g. the members of an `enum` keyword.
func extractSchemaStringArray(pairNode *tree_sitter.Node, source []byte) []string {
	var values []string
	for i := uint(0); i < pairNode.ChildCount(); i++ {
		child := pairNode.Child(i)
		if child.Kind() != "array" {
			continue
		}
		for j := uint(0); j < child.ChildCount(); j++ {
			item := child.Child(j)
			if item.Kind() == "string" {
				values = append(values, unquoteJSString(item.Utf8Text(source)))
			}
		}
	}
	return values
}

func findOptionsObject(argsNode *tree_sitter.Node, source []byte) *tree_sitter.Node {
	pathFound := false

	for i := uint(0); i < argsNode.ChildCount(); i++ {
		child := argsNode.Child(i)
		if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
			continue
		}

		if !pathFound {
			if child.Kind() == "string" {
				pathFound = true
			}
			continue
		}

		if child.Kind() == "object" {
			return child
		}

		return nil
	}

	return nil
}

func extractSchemaFromOptions(optionsNode *tree_sitter.Node, source []byte) *tree_sitter.Node {
	for i := uint(0); i < optionsNode.ChildCount(); i++ {
		child := optionsNode.Child(i)
		if child.Kind() != "pair" {
			continue
		}

		key := extractPairKey(child, source)
		if key == "schema" {
			for j := uint(0); j < child.ChildCount(); j++ {
				pairChild := child.Child(j)
				if pairChild.Kind() == "object" {
					return pairChild
				}
			}
		}
	}

	return nil
}

func extractSchemaFromRouteObject(objNode *tree_sitter.Node, source []byte) *tree_sitter.Node {
	for i := uint(0); i < objNode.ChildCount(); i++ {
		child := objNode.Child(i)
		if child.Kind() != "pair" {
			continue
		}

		key := extractPairKey(child, source)
		if key == "schema" {
			for j := uint(0); j < child.ChildCount(); j++ {
				pairChild := child.Child(j)
				if pairChild.Kind() == "object" {
					return pairChild
				}
			}
		}
	}

	return nil
}

func extractSchemaBody(schemaNode *tree_sitter.Node, source []byte, unresolved *collector.UnresolvedSet) *model.ObjectModel {
	for i := uint(0); i < schemaNode.ChildCount(); i++ {
		child := schemaNode.Child(i)
		if child.Kind() != "pair" {
			continue
		}

		key := extractPairKey(child, source)
		if key == "body" {
			return extractSchemaValue(child, source, unresolved)
		}
	}

	return nil
}

func extractSchemaQuery(schemaNode *tree_sitter.Node, source []byte, unresolved *collector.UnresolvedSet) *model.ObjectModel {
	for i := uint(0); i < schemaNode.ChildCount(); i++ {
		child := schemaNode.Child(i)
		if child.Kind() != "pair" {
			continue
		}

		key := extractPairKey(child, source)
		if key == "querystring" || key == "query" {
			return extractSchemaValue(child, source, unresolved)
		}
	}

	return nil
}

func extractSchemaParams(schemaNode *tree_sitter.Node, source []byte, unresolved *collector.UnresolvedSet) *model.ObjectModel {
	for i := uint(0); i < schemaNode.ChildCount(); i++ {
		child := schemaNode.Child(i)
		if child.Kind() != "pair" {
			continue
		}

		key := extractPairKey(child, source)
		if key == "params" {
			return extractSchemaValue(child, source, unresolved)
		}
	}

	return nil
}

func extractSchemaResponse(schemaNode *tree_sitter.Node, source []byte, unresolved *collector.UnresolvedSet) *model.ObjectModel {
	for i := uint(0); i < schemaNode.ChildCount(); i++ {
		child := schemaNode.Child(i)
		if child.Kind() != "pair" {
			continue
		}

		key := extractPairKey(child, source)
		if key == "response" {
			for j := uint(0); j < child.ChildCount(); j++ {
				pairChild := child.Child(j)
				if pairChild.Kind() == "object" {
					return extractFirstResponseSchema(pairChild, source, unresolved)
				}
			}
		}
	}

	return nil
}

func extractFirstResponseSchema(responseNode *tree_sitter.Node, source []byte, unresolved *collector.UnresolvedSet) *model.ObjectModel {
	for i := uint(0); i < responseNode.ChildCount(); i++ {
		child := responseNode.Child(i)
		if child.Kind() != "pair" {
			continue
		}

		key := extractPairKey(child, source)
		if key == "200" || key == "201" || key == "default" {
			schema := extractSchemaValue(child, source, unresolved)
			if schema != nil {
				return schema
			}
		}
	}

	for i := uint(0); i < responseNode.ChildCount(); i++ {
		child := responseNode.Child(i)
		if child.Kind() != "pair" {
			continue
		}

		schema := extractSchemaValue(child, source, unresolved)
		if schema != nil {
			return schema
		}
	}

	return nil
}

func extractSchemaValue(pairNode *tree_sitter.Node, source []byte, unresolved *collector.UnresolvedSet) *model.ObjectModel {
	for i := uint(0); i < pairNode.ChildCount(); i++ {
		child := pairNode.Child(i)
		if child.Kind() == "object" {
			return parseJSONSchemaObject(child, source, unresolved)
		}
		if child.Kind() == "identifier" {
			return model.SingleModel(child.Utf8Text(source))
		}
	}

	return nil
}

func parseJSONSchemaObject(node *tree_sitter.Node, source []byte, unresolved *collector.UnresolvedSet) *model.ObjectModel {
	schemaType := ""
	var propertiesNode *tree_sitter.Node
	var itemsNode *tree_sitter.Node
	requiredFields := make(map[string]bool)

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() != "pair" {
			continue
		}

		key := extractPairKey(child, source)

		switch key {
		case "type":
			schemaType = extractPairStringValue(child, source)
		case "properties":
			for j := uint(0); j < child.ChildCount(); j++ {
				pairChild := child.Child(j)
				if pairChild.Kind() == "object" {
					propertiesNode = pairChild
				}
			}
		case "items":
			for j := uint(0); j < child.ChildCount(); j++ {
				pairChild := child.Child(j)
				if pairChild.Kind() == "object" {
					itemsNode = pairChild
				}
			}
		case "required":
			requiredFields = extractRequiredArray(child, source)
		}
	}

	switch schemaType {
	case "string":
		return model.SingleModel(model.JsonTypeString)
	case "integer":
		return model.SingleModel(model.JsonTypeInt)
	case "number":
		return model.SingleModel(model.JsonTypeDouble)
	case "boolean":
		return model.SingleModel(model.JsonTypeBoolean)
	case "null":
		return model.SingleModel(model.JsonTypeNull)
	case "array":
		if itemsNode != nil {
			itemModel := parseJSONSchemaObject(itemsNode, source, unresolved)
			return model.ArrayModel(itemModel)
		}
		return model.ArrayModel(model.NullModel())
	case "object":
		if propertiesNode != nil {
			return parseJSONSchemaProperties(propertiesNode, source, requiredFields, unresolved)
		}
		return model.EmptyObject()
	default:
		if propertiesNode != nil {
			return parseJSONSchemaProperties(propertiesNode, source, requiredFields, unresolved)
		}
		return model.EmptyObject()
	}
}

func parseJSONSchemaProperties(propsNode *tree_sitter.Node, source []byte, requiredFields map[string]bool, unresolved *collector.UnresolvedSet) *model.ObjectModel {
	fields := make(map[string]*model.FieldModel)

	for i := uint(0); i < propsNode.ChildCount(); i++ {
		child := propsNode.Child(i)
		if child.Kind() != "pair" {
			continue
		}

		propName := extractPairKey(child, source)
		if propName == "" {
			continue
		}

		var propModel *model.ObjectModel
		var propSchema *tree_sitter.Node
		for j := uint(0); j < child.ChildCount(); j++ {
			pairChild := child.Child(j)
			if pairChild.Kind() == "object" {
				propSchema = pairChild
				propModel = parseJSONSchemaObject(pairChild, source, unresolved)
				break
			}
		}

		if propModel == nil {
			propType := extractPairStringValue(child, source)
			if propType != "" {
				propModel = mapJSONSchemaType(propType, unresolved)
			} else {
				propModel = model.SingleModel(model.JsonTypeString)
			}
		}

		_, required := requiredFields[propName]
		fm := &model.FieldModel{
			Model:    propModel,
			Required: required,
		}
		if propSchema != nil {
			extractSchemaFieldDoc(propSchema, source).ApplyTo(fm)
		}
		fields[propName] = fm
	}

	if len(fields) == 0 {
		return model.EmptyObject()
	}

	return model.ObjectModelFrom(fields)
}

func extractRequiredArray(pairNode *tree_sitter.Node, source []byte) map[string]bool {
	required := make(map[string]bool)

	for i := uint(0); i < pairNode.ChildCount(); i++ {
		child := pairNode.Child(i)
		if child.Kind() == "array" {
			for j := uint(0); j < child.ChildCount(); j++ {
				arrChild := child.Child(j)
				if arrChild.Kind() == "string" {
					fieldName := unquoteJSString(arrChild.Utf8Text(source))
					required[fieldName] = true
				}
			}
		}
	}

	return required
}

func mapJSONSchemaType(schemaType string, unresolved *collector.UnresolvedSet) *model.ObjectModel {
	switch schemaType {
	case "string":
		return model.SingleModel(model.JsonTypeString)
	case "integer":
		return model.SingleModel(model.JsonTypeInt)
	case "number":
		return model.SingleModel(model.JsonTypeDouble)
	case "boolean":
		return model.SingleModel(model.JsonTypeBoolean)
	case "null":
		return model.SingleModel(model.JsonTypeNull)
	case "array":
		return model.ArrayModel(model.NullModel())
	case "object":
		return model.EmptyObject()
	default:
		unresolved.Record(schemaType)
		return model.SingleModel(schemaType)
	}
}

func extractPairKey(pairNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < pairNode.ChildCount(); i++ {
		child := pairNode.Child(i)
		if child.Kind() == "property_identifier" {
			return child.Utf8Text(source)
		}
		if child.Kind() == "string" {
			return unquoteJSString(child.Utf8Text(source))
		}
		if child.Kind() == "number" {
			return child.Utf8Text(source)
		}
	}

	return ""
}

func extractPairStringValue(pairNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < pairNode.ChildCount(); i++ {
		child := pairNode.Child(i)
		if child.Kind() == "string" {
			return unquoteJSString(child.Utf8Text(source))
		}
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
	}

	return ""
}

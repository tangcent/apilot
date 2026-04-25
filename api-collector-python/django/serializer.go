package django

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type SerializerModel struct {
	Name          string
	Fields        []SerializerField
	EmbeddedTypes []string
	MetaModel     string
	MetaFields    []string
	IsModelSerializer bool
}

type SerializerField struct {
	Name     string
	DRFType  string
	Required bool
	ReadOnly bool
	Many     bool
}

var drfFieldTypeMap = map[string]string{
	"CharField":         "string",
	"TextField":         "string",
	"EmailField":        "string",
	"URLField":          "string",
	"SlugField":         "string",
	"UUIDField":         "string",
	"IPAddressField":    "string",
	"RegexField":        "string",
	"FilePathField":     "string",
	"IntegerField":      "int",
	"SmallIntegerField": "int",
	"BigIntegerField":   "long",
	"PositiveIntegerField":     "int",
	"PositiveSmallIntegerField": "int",
	"FloatField":       "float",
	"DecimalField":     "float",
	"BooleanField":     "boolean",
	"NullBooleanField": "boolean",
	"DateField":        "string",
	"DateTimeField":    "string",
	"TimeField":        "string",
	"DurationField":    "string",
	"FileField":        "string",
	"ImageField":       "string",
	"ListField":        "array",
	"DictField":        "map",
	"JSONField":        "string",
	"SerializerMethodField": "string",
}

func extractSerializers(rootNode *tree_sitter.Node, source []byte) map[string]SerializerModel {
	allClasses := make(map[string]*serializerClassInfo)

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child.Kind() == "class_definition" {
			info := extractSerializerClassInfo(child, source)
			if info != nil {
				allClasses[info.name] = info
			}
		}
	}

	serializerSet := findSerializerClasses(allClasses)

	models := make(map[string]SerializerModel)
	for name, info := range allClasses {
		if !serializerSet[name] {
			continue
		}
		md := SerializerModel{
			Name:              name,
			Fields:            info.fields,
			IsModelSerializer: info.isModelSerializer,
			MetaModel:         info.metaModel,
			MetaFields:        info.metaFields,
		}
		for _, parent := range info.parents {
			if !isDRFBaseClass(parent) {
				md.EmbeddedTypes = append(md.EmbeddedTypes, parent)
			}
		}
		models[name] = md
	}

	return models
}

type serializerClassInfo struct {
	name              string
	parents           []string
	fields            []SerializerField
	isModelSerializer bool
	metaModel         string
	metaFields        []string
}

func extractSerializerClassInfo(node *tree_sitter.Node, source []byte) *serializerClassInfo {
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

	info := &serializerClassInfo{name: name}

	if argList != nil {
		info.parents = extractParentClassNames(argList, source)
		for _, p := range info.parents {
			if isModelSerializerBase(p) {
				info.isModelSerializer = true
			}
		}
	}

	if body != nil {
		info.fields = extractSerializerFields(body, source)
		info.metaModel, info.metaFields = extractMetaInfo(body, source)
	}

	return info
}

func extractParentClassNames(argList *tree_sitter.Node, source []byte) []string {
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

func isModelSerializerBase(parent string) bool {
	return parent == "ModelSerializer" ||
		strings.HasSuffix(parent, ".ModelSerializer") ||
		parent == "HyperlinkedModelSerializer" ||
		strings.HasSuffix(parent, ".HyperlinkedModelSerializer")
}

func isDRFBaseClass(parent string) bool {
	switch parent {
	case "Serializer", "ModelSerializer", "HyperlinkedModelSerializer",
		"HyperlinkedRelatedField", "StringRelatedField", "PrimaryKeyRelatedField",
		"SlugRelatedField", "ListSerializer":
		return true
	}
	if strings.HasSuffix(parent, ".Serializer") ||
		strings.HasSuffix(parent, ".ModelSerializer") ||
		strings.HasSuffix(parent, ".HyperlinkedModelSerializer") ||
		strings.HasSuffix(parent, ".ListSerializer") {
		return true
	}
	return false
}

func extractSerializerFields(body *tree_sitter.Node, source []byte) []SerializerField {
	var fields []SerializerField

	for i := uint(0); i < body.ChildCount(); i++ {
		child := body.Child(i)
		if child.Kind() != "expression_statement" {
			continue
		}

		for j := uint(0); j < child.ChildCount(); j++ {
			subChild := child.Child(j)
			if subChild.Kind() == "assignment" {
				f := extractSerializerFieldFromAssignment(subChild, source)
				if f != nil {
					fields = append(fields, *f)
				}
			}
		}
	}

	return fields
}

func extractSerializerFieldFromAssignment(node *tree_sitter.Node, source []byte) *SerializerField {
	var name string
	var drfType string
	var required bool = true
	var readOnly bool
	var many bool
	leftFound := false

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "identifier":
			if !leftFound {
				name = child.Utf8Text(source)
				leftFound = true
			}
		case "=":
		case "call":
			if leftFound {
				drfType, required, readOnly, many = extractDRFFieldCall(child, source)
			}
		}
	}

	if name == "" || drfType == "" {
		return nil
	}

	if name == "serializer_class" {
		return nil
	}

	return &SerializerField{
		Name:     name,
		DRFType:  drfType,
		Required: required,
		ReadOnly: readOnly,
		Many:     many,
	}
}

func extractDRFFieldCall(callNode *tree_sitter.Node, source []byte) (drfType string, required bool, readOnly bool, many bool) {
	required = true

	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "identifier" || child.Kind() == "attribute" {
			drfType = extractDRFTypeName(child.Utf8Text(source))
		}
		if child.Kind() == "argument_list" {
			required, readOnly, many = extractDRFFieldArgs(child, source, required)
		}
	}

	return drfType, required, readOnly, many
}

func extractDRFTypeName(raw string) string {
	parts := strings.Split(raw, ".")
	return parts[len(parts)-1]
}

func extractDRFFieldArgs(argList *tree_sitter.Node, source []byte, defaultRequired bool) (required bool, readOnly bool, many bool) {
	required = defaultRequired

	for i := uint(0); i < argList.ChildCount(); i++ {
		child := argList.Child(i)
		if child.Kind() != "keyword_argument" {
			continue
		}

		var kwName string
		var kwValue string
		for j := uint(0); j < child.ChildCount(); j++ {
			kwChild := child.Child(j)
			if kwChild.Kind() == "identifier" && kwName == "" {
				kwName = kwChild.Utf8Text(source)
			} else if kwChild.Kind() == "=" {
				continue
			} else if kwName != "" && kwValue == "" {
				kwValue = kwChild.Utf8Text(source)
			}
		}

		switch kwName {
		case "required":
			required = kwValue == "True"
		case "read_only":
			readOnly = kwValue == "True"
		case "many":
			many = kwValue == "True"
		case "allow_null":
		case "default":
			if kwValue != "" {
				required = false
			}
		}
	}

	return required, readOnly, many
}

func extractMetaInfo(body *tree_sitter.Node, source []byte) (metaModel string, metaFields []string) {
	for i := uint(0); i < body.ChildCount(); i++ {
		child := body.Child(i)
		if child.Kind() != "class_definition" {
			continue
		}

		var className string
		var classBody *tree_sitter.Node
		for j := uint(0); j < child.ChildCount(); j++ {
			c := child.Child(j)
			if c.Kind() == "identifier" && className == "" {
				className = c.Utf8Text(source)
			}
			if c.Kind() == "block" {
				classBody = c
			}
		}

		if className != "Meta" || classBody == nil {
			continue
		}

		for j := uint(0); j < classBody.ChildCount(); j++ {
			stmt := classBody.Child(j)
			if stmt.Kind() != "expression_statement" {
				continue
			}
			for k := uint(0); k < stmt.ChildCount(); k++ {
				assign := stmt.Child(k)
				if assign.Kind() != "assignment" {
					continue
				}
				var assignName string
				var assignValue string
				for m := uint(0); m < assign.ChildCount(); m++ {
					a := assign.Child(m)
					if a.Kind() == "identifier" && assignName == "" {
						assignName = a.Utf8Text(source)
					} else if a.Kind() == "=" {
						continue
					} else if assignName != "" && assignValue == "" {
						assignValue = a.Utf8Text(source)
					}
				}
				switch assignName {
				case "model":
					metaModel = unquotePythonString(strings.TrimSpace(assignValue))
				case "fields":
					metaFields = extractListLiteralValues(assignValue)
				}
			}
		}
	}

	return metaModel, metaFields
}

func extractListLiteralValues(text string) []string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "[") || !strings.HasSuffix(text, "]") {
		return nil
	}
	inner := text[1 : len(text)-1]
	var values []string
	for _, part := range strings.Split(inner, ",") {
		part = strings.TrimSpace(part)
		part = strings.Trim(part, `"'`)
		if part != "" {
			values = append(values, part)
		}
	}
	return values
}

func findSerializerClasses(allClasses map[string]*serializerClassInfo) map[string]bool {
	serializerSet := make(map[string]bool)

	changed := true
	for changed {
		changed = false
		for name, info := range allClasses {
			if serializerSet[name] {
				continue
			}
			for _, parent := range info.parents {
				if isDRFBaseClass(parent) {
					serializerSet[name] = true
					changed = true
					break
				}
				if serializerSet[parent] {
					serializerSet[name] = true
					changed = true
					break
				}
			}
		}
	}

	return serializerSet
}

func extractSerializerClassFromView(classNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < classNode.ChildCount(); i++ {
		child := classNode.Child(i)
		if child.Kind() != "block" {
			continue
		}
		for j := uint(0); j < child.ChildCount(); j++ {
			stmt := child.Child(j)
			if stmt.Kind() != "expression_statement" {
				continue
			}
			for k := uint(0); k < stmt.ChildCount(); k++ {
				assign := stmt.Child(k)
				if assign.Kind() != "assignment" {
					continue
				}
				var assignName string
				var assignValue string
				for m := uint(0); m < assign.ChildCount(); m++ {
					a := assign.Child(m)
					if a.Kind() == "identifier" && assignName == "" {
						assignName = a.Utf8Text(source)
					} else if a.Kind() == "=" {
						continue
					} else if assignName != "" && assignValue == "" {
						assignValue = a.Utf8Text(source)
					}
				}
				if assignName == "serializer_class" {
					return strings.TrimSpace(assignValue)
				}
			}
		}
	}
	return ""
}

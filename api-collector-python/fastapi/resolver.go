package fastapi

import (
	"fmt"
	"os"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	python "github.com/tree-sitter/tree-sitter-python/bindings/go"

	collector "github.com/tangcent/apilot/api-collector"
	docmeta "github.com/tangcent/apilot/api-docmeta"
	model "github.com/tangcent/apilot/api-model"
)

type PydanticModel struct {
	Name          string
	Fields        []PydanticField
	EmbeddedTypes []string
}

type PydanticField struct {
	Name        string
	Type        string
	Required    bool
	Default     string
	Description string
	Example     string
}

var pythonPrimitives = map[string]string{
	"str":   model.JsonTypeString,
	"int":   model.JsonTypeInt,
	"float": model.JsonTypeFloat,
	"bool":  model.JsonTypeBoolean,
	"bytes": model.JsonTypeString,
	"None":  model.JsonTypeNull,
	"Any":   model.JsonTypeString,
}

var pythonCollectionTypes = map[string]bool{
	"List":      true,
	"Set":       true,
	"FrozenSet": true,
	"Sequence":  true,
	"Tuple":     true,
}

var pythonMapTypes = map[string]bool{
	"Dict":        true,
	"Mapping":     true,
	"OrderedDict": true,
	"DefaultDict": true,
}

func ExtractPydanticModels(rootNode *tree_sitter.Node, source []byte) map[string]PydanticModel {
	allClasses := make(map[string]*classInfo)

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		if child.Kind() == "class_definition" {
			info := extractClassInfo(child, source)
			if info != nil {
				allClasses[info.name] = info
			}
		}
	}

	pydanticSet := findPydanticClasses(allClasses)

	models := make(map[string]PydanticModel)
	for name, info := range allClasses {
		if !pydanticSet[name] {
			continue
		}
		md := PydanticModel{
			Name:   name,
			Fields: info.fields,
		}
		for _, parent := range info.parents {
			if parent != "BaseModel" {
				md.EmbeddedTypes = append(md.EmbeddedTypes, parent)
			}
		}
		models[name] = md
	}

	return models
}

func ExtractPydanticModelsFromFile(filePath string) (map[string]PydanticModel, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	p := tree_sitter.NewParser()
	defer p.Close()

	lang := tree_sitter.NewLanguage(python.Language())
	if err := p.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	tree := p.Parse(source, nil)
	if tree == nil {
		return nil, nil
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	return ExtractPydanticModels(rootNode, source), nil
}

type classInfo struct {
	name    string
	parents []string
	fields  []PydanticField
}

func extractClassInfo(node *tree_sitter.Node, source []byte) *classInfo {
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

	info := &classInfo{name: name}

	if argList != nil {
		info.parents = extractParentClasses(argList, source)
	}

	if body != nil {
		info.fields = extractClassFields(body, source)
	}

	return info
}

func extractParentClasses(argList *tree_sitter.Node, source []byte) []string {
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

func findPydanticClasses(allClasses map[string]*classInfo) map[string]bool {
	pydanticSet := make(map[string]bool)

	changed := true
	for changed {
		changed = false
		for name, info := range allClasses {
			if pydanticSet[name] {
				continue
			}
			for _, parent := range info.parents {
				if parent == "BaseModel" {
					pydanticSet[name] = true
					changed = true
					break
				}
				if pydanticSet[parent] {
					pydanticSet[name] = true
					changed = true
					break
				}
			}
		}
	}

	return pydanticSet
}

func extractClassFields(body *tree_sitter.Node, source []byte) []PydanticField {
	var fields []PydanticField

	for i := uint(0); i < body.ChildCount(); i++ {
		child := body.Child(i)
		if child.Kind() != "expression_statement" {
			continue
		}

		for j := uint(0); j < child.ChildCount(); j++ {
			subChild := child.Child(j)
			if subChild.Kind() == "assignment" {
				f := extractFieldFromAssignment(subChild, source)
				if f != nil {
					fields = append(fields, *f)
				}
			} else if subChild.Kind() == "annotated_assignment" {
				f := extractFieldFromAnnotatedAssignment(subChild, source)
				if f != nil {
					fields = append(fields, *f)
				}
			}
		}
	}

	return fields
}

func extractFieldFromAssignment(node *tree_sitter.Node, source []byte) *PydanticField {
	var name string
	var typeText string
	var required bool = true
	var defaultVal string
	var description string
	var example string

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
			required = false
		case "call":
			if leftFound {
				info := extractFieldTypeFromCall(child, source)
				if info.typeName != "" {
					typeText = info.typeName
				}
				defaultVal = info.defaultVal
				description = info.description
				example = info.example
			}
		case "type":
			typeText = child.Utf8Text(source)
		default:
			if leftFound && typeText == "" && defaultVal == "" {
				defaultVal = child.Utf8Text(source)
			}
		}
	}

	if name == "" {
		return nil
	}

	return &PydanticField{
		Name:        name,
		Type:        typeText,
		Required:    required,
		Default:     defaultVal,
		Description: description,
		Example:     example,
	}
}

func extractFieldFromAnnotatedAssignment(node *tree_sitter.Node, source []byte) *PydanticField {
	var name string
	var typeText string
	var required bool = true
	var defaultVal string
	var description string
	var example string

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "identifier":
			name = child.Utf8Text(source)
		case "type":
			typeText = child.Utf8Text(source)
		case "=":
			required = false
		case "call":
			info := extractFieldTypeFromCall(child, source)
			if info.typeName == "" {
				defaultVal = info.defaultVal
				description = info.description
				example = info.example
			} else if name != "" && typeText != "" {
				defaultVal = child.Utf8Text(source)
			}
		default:
			if child.Kind() != ":" && name != "" && typeText != "" {
				defaultVal = child.Utf8Text(source)
			}
		}
	}

	if name == "" || typeText == "" {
		return nil
	}

	return &PydanticField{
		Name:        name,
		Type:        typeText,
		Required:    required,
		Default:     defaultVal,
		Description: description,
		Example:     example,
	}
}

// fieldCallInfo carries the metadata found in a field's right-hand-side call
// expression, e.g. `Field(default="x", description="user name")`.
type fieldCallInfo struct {
	// typeName is the callee when it doubles as the field type. It is empty
	// for descriptor calls such as Field(...), whose type comes from the
	// annotation instead.
	typeName    string
	defaultVal  string
	description string
	example     string
}

// isPydanticFieldCall reports whether a callee names the pydantic Field
// descriptor rather than a type constructor. Any qualified form is accepted,
// so `Field`, `pydantic.Field` and `pydantic.v1.Field` all match.
func isPydanticFieldCall(callee string) bool {
	return callee == "Field" || strings.HasSuffix(callee, ".Field")
}

func extractFieldTypeFromCall(callNode *tree_sitter.Node, source []byte) fieldCallInfo {
	var info fieldCallInfo
	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "identifier" || child.Kind() == "attribute" {
			info.typeName = child.Utf8Text(source)
		}
		if child.Kind() == "argument_list" {
			info = extractFieldTypeInfoFromArgs(child, source, info)
		}
	}
	return info
}

func extractFieldTypeInfoFromArgs(argList *tree_sitter.Node, source []byte, info fieldCallInfo) fieldCallInfo {
	callee := info.typeName
	if isPydanticFieldCall(callee) {
		// Field(...) is a descriptor, not the field's type.
		info.typeName = ""
	}
	for i := uint(0); i < argList.ChildCount(); i++ {
		child := argList.Child(i)
		if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
			continue
		}
		if child.Kind() == "ellipsis" {
			continue
		}
		if child.Kind() == "keyword_argument" {
			key, value := keywordArgumentParts(child, source)
			switch key {
			case "default", "default_factory":
				info.defaultVal = "has_default"
			case "description":
				info.description = unquoteStringLiteral(value)
			case "example":
				info.example = unquoteStringLiteral(value)
			}
			continue
		}
		if isPydanticFieldCall(callee) {
			text := child.Utf8Text(source)
			if strings.HasPrefix(text, `"`) || strings.HasPrefix(text, `'`) {
				info.defaultVal = text
			}
		}
	}
	return info
}

// keywordArgumentParts splits a keyword_argument node into the keyword name
// and the raw text of its value.
func keywordArgumentParts(node *tree_sitter.Node, source []byte) (key string, value string) {
	isValue := false
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch {
		case child.Kind() == "identifier" && !isValue:
			key = child.Utf8Text(source)
		case child.Kind() == "=":
			isValue = true
		case isValue:
			value = strings.TrimSpace(child.Utf8Text(source))
		}
	}
	return key, value
}

// unquoteStringLiteral strips the quotes around a Python string literal.
// Non-literal text is returned unchanged.
func unquoteStringLiteral(text string) string {
	text = strings.TrimSpace(text)
	if len(text) >= 6 &&
		((strings.HasPrefix(text, `"""`) && strings.HasSuffix(text, `"""`)) ||
			(strings.HasPrefix(text, "'''") && strings.HasSuffix(text, "'''"))) {
		return text[3 : len(text)-3]
	}
	if len(text) >= 2 && (text[0] == '"' || text[0] == '\'') && text[len(text)-1] == text[0] {
		return text[1 : len(text)-1]
	}
	return text
}

type PythonTypeResolver struct {
	modelRegistry     map[string]PydanticModel
	resolving         map[string]bool
	dependencyResolver collector.DependencyResolver
	unresolved         *collector.UnresolvedSet
}

func NewPythonTypeResolver(models map[string]PydanticModel) *PythonTypeResolver {
	return &PythonTypeResolver{
		modelRegistry: models,
		resolving:     make(map[string]bool),
	}
}

func (r *PythonTypeResolver) SetDependencyResolver(dr collector.DependencyResolver) {
	r.dependencyResolver = dr
}

// SetUnresolved attaches a shared sink that records type names this resolver
// could not expand. It is optional; without it Resolve behaves as before.
func (r *PythonTypeResolver) SetUnresolved(u *collector.UnresolvedSet) {
	r.unresolved = u
}

// Unresolved returns the type names this resolver failed to resolve, mapped to
// their occurrence counts.
func (r *PythonTypeResolver) Unresolved() map[string]int {
	return r.unresolved.Counts()
}

// recordUnresolved notes that typeName could not be expanded into fields and
// was rendered as an opaque single value.
func (r *PythonTypeResolver) recordUnresolved(typeName string) {
	r.unresolved.Record(typeName)
}

func (r *PythonTypeResolver) Resolve(typeText string) *model.ObjectModel {
	if typeText == "" {
		return model.NullModel()
	}

	typeText = strings.TrimSpace(typeText)

	if jsonType, ok := pythonPrimitives[typeText]; ok {
		return model.SingleModel(jsonType)
	}

	if typeText == "NoneType" {
		return model.NullModel()
	}

	baseName, typeArgs := ParsePythonGenericType(typeText)

	// `Generic[T]` is a typing declaration used as a base class
	// (`class PageResult(BaseModel, Generic[T])`), not a field type. It carries
	// no fields of its own, so it is neither expandable nor a failure.
	if baseName == "Generic" {
		return model.NullModel()
	}

	if baseName == "Optional" {
		if len(typeArgs) > 0 {
			inner := r.Resolve(typeArgs[0])
			return inner
		}
		return model.NullModel()
	}

	if baseName == "Union" {
		nonNoneArgs := make([]string, 0, len(typeArgs))
		for _, arg := range typeArgs {
			if arg != "None" && arg != "NoneType" {
				nonNoneArgs = append(nonNoneArgs, arg)
			}
		}
		if len(nonNoneArgs) == 1 {
			return r.Resolve(nonNoneArgs[0])
		}
		if len(nonNoneArgs) > 1 {
			joined := strings.Join(nonNoneArgs, " | ")
			r.recordUnresolved(joined)
			return model.SingleModel(joined)
		}
		return model.NullModel()
	}

	if pythonCollectionTypes[baseName] {
		if len(typeArgs) > 0 {
			itemModel := r.Resolve(typeArgs[0])
			return model.ArrayModel(itemModel)
		}
		return model.ArrayModel(model.NullModel())
	}

	if baseName == "list" && len(typeArgs) > 0 {
		itemModel := r.Resolve(typeArgs[0])
		return model.ArrayModel(itemModel)
	}

	if baseName == "set" && len(typeArgs) > 0 {
		itemModel := r.Resolve(typeArgs[0])
		return model.ArrayModel(itemModel)
	}

	if baseName == "tuple" && len(typeArgs) > 0 {
		itemModel := r.Resolve(typeArgs[0])
		return model.ArrayModel(itemModel)
	}

	if pythonMapTypes[baseName] {
		keyModel := model.SingleModel(model.JsonTypeString)
		valueModel := model.NullModel()
		if len(typeArgs) >= 2 {
			valueModel = r.Resolve(typeArgs[1])
		} else if len(typeArgs) == 1 {
			valueModel = r.Resolve(typeArgs[0])
		}
		return model.MapModel(keyModel, valueModel)
	}

	if baseName == "dict" {
		keyModel := model.SingleModel(model.JsonTypeString)
		valueModel := model.NullModel()
		if len(typeArgs) >= 2 {
			valueModel = r.Resolve(typeArgs[1])
		} else if len(typeArgs) == 1 {
			valueModel = r.Resolve(typeArgs[0])
		}
		return model.MapModel(keyModel, valueModel)
	}

	md, found := r.modelRegistry[baseName]
	if found {
		if r.resolving[baseName] {
			return model.RefModel(baseName)
		}

		r.resolving[baseName] = true
		defer func() { delete(r.resolving, baseName) }()

		fields := make(map[string]*model.FieldModel, len(md.Fields))
		for _, f := range md.Fields {
			fieldModel := r.resolveFieldModel(f)
			fields[f.Name] = fieldModel
		}

		for _, embedded := range md.EmbeddedTypes {
			embeddedModel := r.Resolve(embedded)
			if embeddedModel != nil && embeddedModel.IsObject() {
				for k, v := range embeddedModel.Fields {
					if _, exists := fields[k]; !exists {
						fields[k] = v
					}
				}
			}
		}

		return &model.ObjectModel{
			Kind:     model.KindObject,
			TypeName: baseName,
			Fields:   fields,
		}
	}

	if r.dependencyResolver != nil {
		if rt := r.dependencyResolver.ResolveType(baseName); rt != nil {
			md := resolvedTypeToPydanticModel(rt)
			r.modelRegistry[baseName] = md
			return r.Resolve(typeText)
		}
	}

	r.recordUnresolved(typeText)
	return model.SingleModel(typeText)
}

func resolvedTypeToPydanticModel(rt *collector.ResolvedType) PydanticModel {
	md := PydanticModel{
		Name: rt.Name,
	}
	for _, f := range rt.Fields {
		md.Fields = append(md.Fields, PydanticField{
			Name:     f.Name,
			Type:     f.Type,
			Required: f.Required,
		})
	}
	for _, iface := range rt.Interfaces {
		md.EmbeddedTypes = append(md.EmbeddedTypes, iface)
	}
	return md
}

func (r *PythonTypeResolver) resolveFieldModel(f PydanticField) *model.FieldModel {
	var fieldModel *model.ObjectModel
	if f.Type != "" {
		fieldModel = r.Resolve(f.Type)
	} else {
		fieldModel = model.SingleModel(model.JsonTypeString)
	}

	fm := &model.FieldModel{
		Model:        fieldModel,
		Required:     f.Required,
		DefaultValue: f.Default,
	}
	docmeta.Documentation{Comment: f.Description, Demo: f.Example}.ApplyTo(fm)
	return fm
}

func ParsePythonGenericType(typeText string) (string, []string) {
	idx := strings.Index(typeText, "[")
	if idx == -1 {
		return typeText, nil
	}

	baseName := typeText[:idx]
	rest := typeText[idx:]

	if len(rest) < 2 || rest[0] != '[' || rest[len(rest)-1] != ']' {
		return typeText, nil
	}

	inner := rest[1 : len(rest)-1]
	args := splitPythonTypeArgs(inner)
	return baseName, args
}

func splitPythonTypeArgs(s string) []string {
	var args []string
	depth := 0
	start := 0

	for i, c := range s {
		switch c {
		case '[':
			depth++
		case ']':
			depth--
		case ',':
			if depth == 0 {
				arg := strings.TrimSpace(s[start:i])
				if arg != "" {
					args = append(args, arg)
				}
				start = i + 1
			}
		}
	}

	if start < len(s) {
		arg := strings.TrimSpace(s[start:])
		if arg != "" {
			args = append(args, arg)
		}
	}

	return args
}

func extractResponseModelFromDecorator(decorator *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < decorator.ChildCount(); i++ {
		child := decorator.Child(i)
		if child.Kind() == "call" {
			return extractResponseModelFromCall(child, source)
		}
	}
	return ""
}

func extractResponseModelFromCall(callNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "argument_list" {
			return extractResponseModelFromArgs(child, source)
		}
	}
	return ""
}

func extractResponseModelFromArgs(argList *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < argList.ChildCount(); i++ {
		child := argList.Child(i)
		if child.Kind() == "keyword_argument" {
			var kwName string
			var kwValue string
			for j := uint(0); j < child.ChildCount(); j++ {
				kwChild := child.Child(j)
				if kwChild.Kind() == "identifier" && kwName == "" {
					kwName = kwChild.Utf8Text(source)
				} else if kwChild.Kind() != "=" && kwName != "" && kwValue == "" {
					kwValue = kwChild.Utf8Text(source)
				}
			}
			if kwName == "response_model" && kwValue != "" {
				return kwValue
			}
		}
	}
	return ""
}

func extractReturnType(funcDef *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < funcDef.ChildCount(); i++ {
		child := funcDef.Child(i)
		if child.Kind() == "type" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

func resolveParamType(p funcParam, typeResolver *PythonTypeResolver) funcParam {
	if p.typ != "" && p.typ != "text" && p.typ != "file" {
		return p
	}

	typeText := extractTypeFromFuncParam(p)
	if typeText == "" {
		return p
	}

	resolved := typeResolver.Resolve(typeText)
	if resolved != nil && resolved.IsObject() {
		p.typ = typeText
		p.in = "body"
	} else if resolved != nil && resolved.IsArray() {
		p.typ = typeText
		p.in = "body"
	}

	return p
}

func extractTypeFromFuncParam(p funcParam) string {
	if p.typ != "" && p.typ != "text" && p.typ != "file" {
		return p.typ
	}
	return ""
}

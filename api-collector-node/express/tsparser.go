package express

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

func ParseTSTypes(sourceDir string) (*TSTypeRegistry, error) {
	tsFiles, err := discoverTSFiles(sourceDir)
	if err != nil || len(tsFiles) == 0 {
		return NewTSTypeRegistry(), nil
	}

	ch := make(chan *TSTypeRegistry, len(tsFiles))
	var wg sync.WaitGroup

	for _, path := range tsFiles {
		wg.Add(1)
		go func(filePath string) {
			defer wg.Done()
			registry, err := extractTypesFromFile(filePath)
			if err != nil {
				ch <- NewTSTypeRegistry()
				return
			}
			ch <- registry
		}(path)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	merged := NewTSTypeRegistry()
	for reg := range ch {
		merged.Merge(reg)
	}

	return merged, nil
}

func discoverTSFiles(sourceDir string) ([]string, error) {
	var tsFiles []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".tsx")) {
			tsFiles = append(tsFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tsFiles, nil
}

func extractTypesFromFile(filePath string) (*TSTypeRegistry, error) {
	source, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	p := tree_sitter.NewParser()
	defer p.Close()

	lang := tree_sitter.NewLanguage(typescript.LanguageTypescript())
	if err := p.SetLanguage(lang); err != nil {
		return nil, fmt.Errorf("failed to set language: %w", err)
	}

	tree := p.Parse(source, nil)
	if tree == nil {
		return NewTSTypeRegistry(), nil
	}
	defer tree.Close()

	rootNode := tree.RootNode()
	registry := extractTypesFromAST(rootNode, source)

	return registry, nil
}

func extractTypesFromAST(rootNode *tree_sitter.Node, source []byte) *TSTypeRegistry {
	registry := NewTSTypeRegistry()

	for i := uint(0); i < rootNode.ChildCount(); i++ {
		child := rootNode.Child(i)
		extractTopLevelType(child, registry, source)
	}

	return registry
}

func extractTopLevelType(node *tree_sitter.Node, registry *TSTypeRegistry, source []byte) {
	switch node.Kind() {
	case "interface_declaration":
		iface := extractInterface(node, source)
		if iface != nil {
			registry.Interfaces[iface.Name] = iface
		}
	case "type_alias_declaration":
		alias := extractTypeAlias(node, source)
		if alias != nil {
			registry.TypeAliases[alias.Name] = alias
		}
	case "enum_declaration":
		enum := extractEnum(node, source)
		if enum != nil {
			registry.Enums[enum.Name] = enum
		}
	case "class_declaration":
		iface := extractClassAsInterface(node, source)
		if iface != nil {
			registry.Interfaces[iface.Name] = iface
		}
	case "export_statement":
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			extractTopLevelType(child, registry, source)
		}
	case "ambient_declaration":
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			extractTopLevelType(child, registry, source)
		}
	case "declaration_list":
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			extractTopLevelType(child, registry, source)
		}
	}
}

func extractInterface(node *tree_sitter.Node, source []byte) *TSInterface {
	name := ""
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "type_identifier" {
			name = child.Utf8Text(source)
			break
		}
	}
	if name == "" {
		return nil
	}

	iface := &TSInterface{
		Name:    name,
		Comment: extractPrevComment(node, source),
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "type_parameters":
			iface.TypeParameters = extractTypeParameterNames(child, source)
		case "object_type":
			iface.Fields = extractInterfaceFields(child, source)
		case "interface_body":
			iface.Fields = extractInterfaceFields(child, source)
		case "extends_clause":
		}
	}

	return iface
}

func extractClassAsInterface(node *tree_sitter.Node, source []byte) *TSInterface {
	name := ""
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "type_identifier" {
			name = child.Utf8Text(source)
			break
		}
	}
	if name == "" {
		return nil
	}

	iface := &TSInterface{
		Name:    name,
		Comment: extractPrevComment(node, source),
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "type_parameters":
			iface.TypeParameters = extractTypeParameterNames(child, source)
		case "class_body":
			iface.Fields = extractClassFields(child, source)
		case "extends_clause":
		case "implements_clause":
		case "decorator":
		}
	}

	return iface
}

func extractClassFields(classBody *tree_sitter.Node, source []byte) []TSField {
	var fields []TSField

	for i := uint(0); i < classBody.ChildCount(); i++ {
		child := classBody.Child(i)
		if child.Kind() == "public_field_definition" || nodeKindMatchesClassProperty(child) {
			field := extractClassPropertyDefinition(child, source)
			if field != nil {
				fields = append(fields, *field)
			}
		}
	}

	return fields
}

func nodeKindMatchesClassProperty(node *tree_sitter.Node) bool {
	kind := node.Kind()
	return kind == "property_definition" || kind == "field_definition" || kind == "public_field_definition"
}

func extractClassPropertyDefinition(node *tree_sitter.Node, source []byte) *TSField {
	name := ""
	typeStr := ""
	required := true
	comment := ""
	var annotations []string

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "property_identifier", "identifier":
			name = child.Utf8Text(source)
		case "type_annotation":
			typeStr = extractTypeAnnotation(child, source)
		case "?":
			required = false
		case "decorator":
			di := parseClassDecorator(child, source)
			if di != nil {
				if di.isOptional {
					required = false
				}
				if di.typeOverride != "" {
					typeStr = di.typeOverride
				}
				if di.description != "" {
					comment = di.description
				}
				if di.name != "" {
					annotations = append(annotations, di.name)
				}
			}
		case "comment":
			if comment == "" {
				comment = cleanComment(child.Utf8Text(source))
			}
		}
	}

	if name == "" {
		return nil
	}

	if typeStr == "" {
		typeStr = "any"
	}

	return &TSField{
		Name:        name,
		Type:        typeStr,
		Required:    required,
		Comment:     comment,
		Annotations: annotations,
	}
}

type classDecoratorInfo struct {
	name         string
	isOptional   bool
	typeOverride string
	description  string
}

var classValidatorOptionalDecorators = map[string]bool{
	"IsOptional": true,
}

var classValidatorTypeDecorators = map[string]string{
	"IsString":  "string",
	"IsNumber":  "number",
	"IsInt":     "number",
	"IsBoolean": "boolean",
	"IsDate":    "string",
	"IsEmail":   "string",
	"IsArray":   "any[]",
	"IsObject":  "object",
	"IsEnum":    "string",
}

func parseClassDecorator(decoratorNode *tree_sitter.Node, source []byte) *classDecoratorInfo {
	info := &classDecoratorInfo{}

	callNode := findChildByKindTSParser(decoratorNode, "call_expression")
	if callNode != nil {
		name := extractDecoratorNameFromCall(callNode, source)
		info.name = name

		if classValidatorOptionalDecorators[name] {
			info.isOptional = true
		}

		if typeName, ok := classValidatorTypeDecorators[name]; ok && typeName != "" {
			info.typeOverride = typeName
		}

		if name == "ApiProperty" {
			info = parseApiPropertyArgs(callNode, source, info)
		}

		return info
	}

	for i := uint(0); i < decoratorNode.ChildCount(); i++ {
		child := decoratorNode.Child(i)
		if child.Kind() == "identifier" {
			decoratorName := child.Utf8Text(source)
			info.name = decoratorName

			if classValidatorOptionalDecorators[decoratorName] {
				info.isOptional = true
			}
			if typeName, ok := classValidatorTypeDecorators[decoratorName]; ok {
				info.typeOverride = typeName
			}
			return info
		}
	}

	return info
}

func extractDecoratorNameFromCall(callNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < callNode.ChildCount(); i++ {
		child := callNode.Child(i)
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
		if child.Kind() == "member_expression" {
			for j := uint(0); j < child.ChildCount(); j++ {
				memberChild := child.Child(j)
				if memberChild.Kind() == "property_identifier" {
					return memberChild.Utf8Text(source)
				}
			}
		}
	}
	return ""
}

func parseApiPropertyArgs(callNode *tree_sitter.Node, source []byte, info *classDecoratorInfo) *classDecoratorInfo {
	argsNode := findChildByKindTSParser(callNode, "arguments")
	if argsNode == nil {
		return info
	}

	for i := uint(0); i < argsNode.ChildCount(); i++ {
		child := argsNode.Child(i)
		if child.Kind() == "object" {
			info = extractApiPropertyObject(child, source, info)
			break
		}
	}

	return info
}

func extractApiPropertyObject(objNode *tree_sitter.Node, source []byte, info *classDecoratorInfo) *classDecoratorInfo {
	for i := uint(0); i < objNode.ChildCount(); i++ {
		child := objNode.Child(i)
		if child.Kind() != "pair" {
			continue
		}

		key := extractPairKeyTSParser(child, source)
		switch key {
		case "description":
			val := extractPairStringValueTSParser(child, source)
			if val != "" {
				info.description = val
			}
		case "type":
			val := extractPairStringValueTSParser(child, source)
			if val != "" {
				info.typeOverride = mapSwaggerTypeToTS(val)
			}
		case "required":
			for j := uint(0); j < child.ChildCount(); j++ {
				pairChild := child.Child(j)
				if pairChild.Kind() == "true" {
					info.isOptional = false
				} else if pairChild.Kind() == "false" {
					info.isOptional = true
				}
			}
		case "example":
		case "enum":
		}
	}

	return info
}

func mapSwaggerTypeToTS(swaggerType string) string {
	switch swaggerType {
	case "string":
		return "string"
	case "number", "integer":
		return "number"
	case "boolean":
		return "boolean"
	case "array":
		return "any[]"
	case "object":
		return "object"
	default:
		return swaggerType
	}
}

func extractPairKeyTSParser(pairNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < pairNode.ChildCount(); i++ {
		child := pairNode.Child(i)
		if child.Kind() == "property_identifier" {
			return child.Utf8Text(source)
		}
		if child.Kind() == "string" {
			return unquoteJSStringTSParser(child.Utf8Text(source))
		}
	}
	return ""
}

func extractPairStringValueTSParser(pairNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < pairNode.ChildCount(); i++ {
		child := pairNode.Child(i)
		if child.Kind() == "string" {
			return unquoteJSStringTSParser(child.Utf8Text(source))
		}
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

func unquoteJSStringTSParser(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' {
		return s[1 : len(s)-1]
	}
	return s
}

func findChildByKindTSParser(node *tree_sitter.Node, kind string) *tree_sitter.Node {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == kind {
			return child
		}
	}
	return nil
}

func extractTypeParameterNames(node *tree_sitter.Node, source []byte) []string {
	var names []string
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "type_parameter" {
			for j := uint(0); j < child.ChildCount(); j++ {
				tpChild := child.Child(j)
				if tpChild.Kind() == "type_identifier" {
					names = append(names, tpChild.Utf8Text(source))
					break
				}
			}
		}
	}
	return names
}

func extractInterfaceFields(objectType *tree_sitter.Node, source []byte) []TSField {
	var fields []TSField

	for i := uint(0); i < objectType.ChildCount(); i++ {
		child := objectType.Child(i)
		if child.Kind() == "property_signature" {
			field := extractPropertySignature(child, source)
			if field != nil {
				fields = append(fields, *field)
			}
		}
	}

	return fields
}

func extractPropertySignature(node *tree_sitter.Node, source []byte) *TSField {
	name := ""
	typeStr := ""
	required := true
	comment := ""

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "property_identifier":
			name = child.Utf8Text(source)
		case "type_annotation":
			typeStr = extractTypeAnnotation(child, source)
		case "?":
			required = false
		case "comment":
			comment = cleanComment(child.Utf8Text(source))
		}
	}

	if name == "" {
		return nil
	}

	if typeStr == "" {
		typeStr = "any"
	}

	return &TSField{
		Name:     name,
		Type:     typeStr,
		Required: required,
		Comment:  comment,
	}
}

func extractTypeAnnotation(node *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == ":" {
			continue
		}
		return extractTypeNodeText(child, source)
	}
	return ""
}

func extractTypeNodeText(node *tree_sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	switch node.Kind() {
	case "type_identifier":
		return node.Utf8Text(source)
	case "predefined_type":
		return node.Utf8Text(source)
	case "generic_type":
		return extractGenericTypeText(node, source)
	case "array_type":
		return extractArrayTypeText(node, source)
	case "union_type":
		return extractUnionTypeText(node, source)
	case "intersection_type":
		return extractIntersectionTypeText(node, source)
	case "tuple_type":
		return extractTupleTypeText(node, source)
	case "object_type":
		return "object"
	case "literal_type":
		return node.Utf8Text(source)
	case "parenthesized_type":
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() != "(" && child.Kind() != ")" {
				return extractTypeNodeText(child, source)
			}
		}
		return node.Utf8Text(source)
	case "type_query":
		return node.Utf8Text(source)
	case "conditional_type":
		return node.Utf8Text(source)
	case "indexed_access_type":
		return node.Utf8Text(source)
	case "mapped_type":
		return "object"
	case "optional_type":
		inner := extractInnerType(node, source)
		return inner + "?"
	case "nullable_type":
		inner := extractInnerType(node, source)
		return inner + " | null"
	default:
		return node.Utf8Text(source)
	}
}

func extractInnerType(node *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		text := child.Utf8Text(source)
		if text != "?" && text != "|" && text != "null" {
			return extractTypeNodeText(child, source)
		}
	}
	return ""
}

func extractGenericTypeText(node *tree_sitter.Node, source []byte) string {
	var baseName string
	var args []string

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "type_identifier":
			baseName = child.Utf8Text(source)
		case "type_arguments":
			args = extractTypeArgumentTexts(child, source)
		}
	}

	if len(args) > 0 {
		return baseName + "<" + strings.Join(args, ", ") + ">"
	}
	return baseName
}

func extractTypeArgumentTexts(node *tree_sitter.Node, source []byte) []string {
	var args []string
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "<" || child.Kind() == ">" || child.Kind() == "," {
			continue
		}
		args = append(args, extractTypeNodeText(child, source))
	}
	return args
}

func extractArrayTypeText(node *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() != "[" && child.Kind() != "]" {
			inner := extractTypeNodeText(child, source)
			return inner + "[]"
		}
	}
	return "any[]"
}

func extractUnionTypeText(node *tree_sitter.Node, source []byte) string {
	var parts []string
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "|" {
			continue
		}
		parts = append(parts, extractTypeNodeText(child, source))
	}
	return strings.Join(parts, " | ")
}

func extractIntersectionTypeText(node *tree_sitter.Node, source []byte) string {
	var parts []string
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "&" {
			continue
		}
		parts = append(parts, extractTypeNodeText(child, source))
	}
	return strings.Join(parts, " & ")
}

func extractTupleTypeText(node *tree_sitter.Node, source []byte) string {
	var elems []string
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "[" || child.Kind() == "]" || child.Kind() == "," {
			continue
		}
		elems = append(elems, extractTypeNodeText(child, source))
	}
	return "[" + strings.Join(elems, ", ") + "]"
}

func extractTypeAlias(node *tree_sitter.Node, source []byte) *TSTypeAlias {
	name := ""
	typeDef := ""

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "type_identifier":
			name = child.Utf8Text(source)
		case "type_parameters":
		case "=":
		case "object_type":
			typeDef = "object"
		default:
			if name != "" && child.Kind() != "type_parameters" && child.Kind() != ";" {
				typeDef = extractTypeNodeText(child, source)
			}
		}
	}

	if name == "" {
		return nil
	}

	var typeParams []string
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "type_parameters" {
			typeParams = extractTypeParameterNames(child, source)
			break
		}
	}

	return &TSTypeAlias{
		Name:           name,
		TypeDef:        typeDef,
		TypeParameters: typeParams,
		Comment:        extractPrevComment(node, source),
	}
}

func extractEnum(node *tree_sitter.Node, source []byte) *TSEnum {
	name := ""
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" {
			name = child.Utf8Text(source)
			break
		}
	}
	if name == "" {
		return nil
	}

	enum := &TSEnum{
		Name:    name,
		Comment: extractPrevComment(node, source),
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "enum_body" {
			enum.Members = extractEnumMembers(child, source)
		}
	}

	return enum
}

func extractEnumMembers(node *tree_sitter.Node, source []byte) []TSEnumMember {
	var members []TSEnumMember
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "enum_assignment" {
			member := extractEnumAssignment(child, source)
			if member != nil {
				members = append(members, *member)
			}
		} else if child.Kind() == "property_identifier" {
			members = append(members, TSEnumMember{
				Name:  child.Utf8Text(source),
				Value: "",
			})
		}
	}
	return members
}

func extractEnumAssignment(node *tree_sitter.Node, source []byte) *TSEnumMember {
	name := ""
	value := ""
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "property_identifier":
			name = child.Utf8Text(source)
		case "=":
		case "number", "string", "identifier":
			value = child.Utf8Text(source)
		}
	}
	if name == "" {
		return nil
	}
	return &TSEnumMember{Name: name, Value: value}
}

func extractPrevComment(node *tree_sitter.Node, source []byte) string {
	prev := node.PrevNamedSibling()
	if prev != nil && prev.Kind() == "comment" {
		commentText := prev.Utf8Text(source)
		return cleanComment(commentText)
	}
	return ""
}

func cleanComment(comment string) string {
	if strings.HasPrefix(comment, "/**") {
		return cleanJSDocComment(comment)
	}
	if strings.HasPrefix(comment, "//") {
		return strings.TrimSpace(strings.TrimPrefix(comment, "//"))
	}
	if strings.HasPrefix(comment, "/*") {
		comment = strings.TrimPrefix(comment, "/*")
		comment = strings.TrimSuffix(comment, "*/")
		return strings.TrimSpace(comment)
	}
	return comment
}

func AnalyzeExpressHandler(callNode *tree_sitter.Node, source []byte) *ExpressHandlerInfo {
	info := &ExpressHandlerInfo{}

	handlerNode := findHandlerNode(callNode, source)
	if handlerNode == nil {
		return info
	}

	switch handlerNode.Kind() {
	case "arrow_function":
		info = analyzeArrowHandler(handlerNode, source)
	case "function_expression":
		info = analyzeFunctionHandler(handlerNode, source)
	case "identifier":
		info = analyzeIdentifierHandler(handlerNode, callNode, source)
	}

	return info
}

func findHandlerNode(callNode *tree_sitter.Node, source []byte) *tree_sitter.Node {
	argsNode := findChildByKindTS(callNode, "arguments")
	if argsNode == nil {
		return nil
	}

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
		return child
	}
	return nil
}

func analyzeArrowHandler(node *tree_sitter.Node, source []byte) *ExpressHandlerInfo {
	info := &ExpressHandlerInfo{}
	params := findChildByKindTS(node, "formal_parameters")
	if params == nil {
		return info
	}

	info.ReqBodyType, info.QueryType, info.ParamsType = extractRequestTypes(params, source)

	resTypeFromGeneric := extractResponseTypes(params, source)
	if resTypeFromGeneric != "" {
		info.ResBodyType = resTypeFromGeneric
	} else {
		bodyNode := findChildByKindTS(node, "statement_block")
		if bodyNode != nil {
			info.ResBodyType = extractResponseTypeFromBody(bodyNode, source)
		}
	}

	return info
}

func analyzeFunctionHandler(node *tree_sitter.Node, source []byte) *ExpressHandlerInfo {
	info := &ExpressHandlerInfo{}
	params := findChildByKindTS(node, "formal_parameters")
	if params == nil {
		return info
	}

	info.ReqBodyType, info.QueryType, info.ParamsType = extractRequestTypes(params, source)

	resTypeFromGeneric := extractResponseTypes(params, source)
	if resTypeFromGeneric != "" {
		info.ResBodyType = resTypeFromGeneric
	} else {
		bodyNode := findChildByKindTS(node, "statement_block")
		if bodyNode != nil {
			info.ResBodyType = extractResponseTypeFromBody(bodyNode, source)
		}
	}

	return info
}

func analyzeIdentifierHandler(idNode *tree_sitter.Node, callNode *tree_sitter.Node, source []byte) *ExpressHandlerInfo {
	info := &ExpressHandlerInfo{}

	handlerName := idNode.Utf8Text(source)
	_ = handlerName

	return info
}

func extractRequestTypes(params *tree_sitter.Node, source []byte) (reqBodyType string, queryType string, paramsType string) {
	for i := uint(0); i < params.ChildCount(); i++ {
		child := params.Child(i)
		if child.Kind() == "required_parameter" || child.Kind() == "optional_parameter" {
			paramType := extractParamTypeAnnotation(child, source)
			paramName := extractParamName(child, source)

			if paramName == "req" || paramName == "request" {
				if strings.Contains(paramType, "Request<") {
					reqBodyType, queryType, paramsType = parseExpressRequestGenerics(paramType)
				}
			}
		}
	}
	return
}

func extractResponseTypes(params *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < params.ChildCount(); i++ {
		child := params.Child(i)
		if child.Kind() == "required_parameter" || child.Kind() == "optional_parameter" {
			paramType := extractParamTypeAnnotation(child, source)
			paramName := extractParamName(child, source)

			if paramName == "res" || paramName == "response" {
				if strings.Contains(paramType, "Response<") {
					return parseExpressResponseGenerics(paramType)
				}
			}
		}
	}
	return ""
}

func extractParamTypeAnnotation(paramNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < paramNode.ChildCount(); i++ {
		child := paramNode.Child(i)
		if child.Kind() == "type_annotation" {
			return extractTypeAnnotation(child, source)
		}
	}
	return ""
}

func extractParamName(paramNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < paramNode.ChildCount(); i++ {
		child := paramNode.Child(i)
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

func parseExpressRequestGenerics(requestType string) (reqBodyType string, queryType string, paramsType string) {
	if !strings.Contains(requestType, "<") || !strings.Contains(requestType, ">") {
		return "", "", ""
	}

	start := strings.Index(requestType, "<")
	end := strings.LastIndex(requestType, ">")
	if start >= end {
		return "", "", ""
	}

	inner := requestType[start+1 : end]
	args := splitTypeArgs(inner)

	if len(args) >= 1 {
		paramsType = strings.TrimSpace(args[0])
		if paramsType == "ParamsDictionary" || paramsType == "Record<string, string>" || paramsType == "{}" {
			paramsType = ""
		}
	}
	if len(args) >= 2 {
		resBody := strings.TrimSpace(args[1])
		_ = resBody
	}
	if len(args) >= 3 {
		reqBodyType = strings.TrimSpace(args[2])
		if reqBodyType == "any" || reqBodyType == "{}" {
			reqBodyType = ""
		}
	}
	if len(args) >= 4 {
		queryType = strings.TrimSpace(args[3])
		if queryType == "qs.ParsedQs" || queryType == "ParsedQs" {
			queryType = ""
		}
	}

	return reqBodyType, queryType, paramsType
}

func parseExpressResponseGenerics(responseType string) string {
	if !strings.Contains(responseType, "<") || !strings.Contains(responseType, ">") {
		return ""
	}

	start := strings.Index(responseType, "<")
	end := strings.LastIndex(responseType, ">")
	if start >= end {
		return ""
	}

	inner := responseType[start+1 : end]
	args := splitTypeArgs(inner)

	if len(args) >= 1 {
		bodyType := strings.TrimSpace(args[0])
		if bodyType != "" && bodyType != "any" && bodyType != "{}" {
			return bodyType
		}
	}

	return ""
}

func extractResponseTypeFromBody(bodyNode *tree_sitter.Node, source []byte) string {
	var resType string
	walkForResJson(bodyNode, source, &resType)
	return resType
}

func walkForResJson(node *tree_sitter.Node, source []byte, resType *string) bool {
	if node == nil {
		return false
	}

	if node.Kind() == "call_expression" {
		callee := findChildByKindTS(node, "member_expression")
		if callee != nil {
			prop := findChildByKindTS(callee, "property_identifier")
			if prop != nil && prop.Utf8Text(source) == "json" {
				argsNode := findChildByKindTS(node, "arguments")
				if argsNode != nil {
					*resType = inferTypeFromArguments(argsNode, source)
					return true
				}
			}
		}
	}

	for i := uint(0); i < node.ChildCount(); i++ {
		if walkForResJson(node.Child(i), source, resType) {
			return true
		}
	}

	return false
}

func inferTypeFromArguments(argsNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < argsNode.ChildCount(); i++ {
		child := argsNode.Child(i)
		if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
			continue
		}
		return inferTypeFromExpression(child, source)
	}
	return ""
}

func inferTypeFromExpression(node *tree_sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}

	switch node.Kind() {
	case "object":
		return inferTypeFromObjectLiteral(node, source)
	case "array":
		return "any[]"
	case "string":
		return "string"
	case "number":
		return "number"
	case "true", "false":
		return "boolean"
	case "null":
		return "null"
	case "identifier":
		return node.Utf8Text(source)
	case "call_expression":
		return ""
	case "member_expression":
		return ""
	case "parenthesized_expression":
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() != "(" && child.Kind() != ")" {
				return inferTypeFromExpression(child, source)
			}
		}
		return ""
	default:
		return ""
	}
}

func inferTypeFromObjectLiteral(node *tree_sitter.Node, source []byte) string {
	fields := make(map[string]string)

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "pair" {
			key, valType := extractPairType(child, source)
			if key != "" {
				fields[key] = valType
			}
		} else if child.Kind() == "shorthand_property_identifier" {
			fields[child.Utf8Text(source)] = "any"
		} else if child.Kind() == "spread_element" {
		} else if child.Kind() == "method_definition" {
		}
	}

	if len(fields) == 0 {
		return "object"
	}

	return "object"
}

func extractPairType(node *tree_sitter.Node, source []byte) (string, string) {
	key := ""
	valType := "any"

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		kind := child.Kind()
		if kind == "property_identifier" || kind == "string_fragment" {
			if key == "" {
				key = child.Utf8Text(source)
			}
		} else if kind == "string" {
			if key == "" {
				key = unquoteJSString(child.Utf8Text(source))
			} else {
				valType = "string"
			}
		} else if kind == ":" {
		} else if kind == "number" {
			valType = "number"
		} else if kind == "true" || kind == "false" {
			valType = "boolean"
		} else if kind == "null" {
			valType = "null"
		} else if kind == "array" {
			valType = "any[]"
		} else if kind == "object" {
			valType = "object"
		} else if kind == "identifier" {
			valType = "any"
		} else {
			inferred := inferTypeFromExpression(child, source)
			if inferred != "" {
				valType = inferred
			}
		}
	}

	return key, valType
}

func findChildByKindTS(node *tree_sitter.Node, kind string) *tree_sitter.Node {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == kind {
			return child
		}
	}
	return nil
}

func splitTypeArgs(s string) []string {
	var args []string
	depth := 0
	start := 0

	for i, c := range s {
		switch c {
		case '<':
			depth++
		case '>':
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

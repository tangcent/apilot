package parser

import (
	"strconv"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

func extractFile(root *tree_sitter.Node, source []byte, filePath string, result *ParseResult) {
	packageName := extractPackageName(root, source)
	imports := extractImports(root, source)

	var visit func(node *tree_sitter.Node)
	visit = func(node *tree_sitter.Node) {
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			switch child.Kind() {
			case "class_declaration", "object_declaration":
				if result.Error != nil {
					return
				}
				if err := extractClass(child, source, packageName, filePath, imports, result); err != nil {
					result.Error = err
					return
				}
			case "annotated_expression":
				// A declaration can be wrapped in annotation expressions at
				// file scope. Descend so the class is not missed.
				visit(child)
			case "typealias":
				result.Warnings = append(result.Warnings,
					filePath+": typealias declarations are not expanded; references to the alias stay unresolved")
			}
		}
	}
	visit(root)
}

func extractPackageName(root *tree_sitter.Node, source []byte) string {
	header := firstChildOfKind(root, "package_header")
	if header == nil {
		return ""
	}
	if id := firstChildOfKind(header, "qualified_identifier", "identifier"); id != nil {
		return id.Utf8Text(source)
	}
	text := strings.TrimSpace(header.Utf8Text(source))
	return strings.TrimSpace(strings.TrimPrefix(text, "package"))
}

func extractImports(root *tree_sitter.Node, source []byte) []Import {
	var imports []Import
	for i := uint(0); i < root.ChildCount(); i++ {
		child := root.Child(i)
		if child.Kind() != "import" {
			continue
		}
		text := strings.TrimSpace(child.Utf8Text(source))
		text = strings.TrimSpace(strings.TrimPrefix(text, "import"))
		if idx := strings.Index(text, " as "); idx >= 0 {
			text = strings.TrimSpace(text[:idx])
		}
		if text == "" {
			continue
		}
		if strings.HasSuffix(text, ".*") {
			imports = append(imports, Import{Path: strings.TrimSuffix(text, ".*"), OnDemand: true})
			continue
		}
		imports = append(imports, Import{Path: text})
	}
	return imports
}

func extractClass(node *tree_sitter.Node, source []byte, packageName, filePath string, imports []Import, result *ParseResult) error {
	modifiers := firstChildOfKind(node, "modifiers")
	anns, flags, err := parseModifiers(modifiers, source, filePath)
	if err != nil {
		return err
	}

	leading, kdoc, err := leadingAnnotations(node, source, filePath)
	if err != nil {
		return err
	}
	anns = mergeAnnotations(anns, leading)
	if kdoc == "" {
		kdoc = leadingKDoc(node, source)
	}

	nameNode := firstChildOfKind(node, "identifier")
	if nameNode == nil {
		return nil
	}

	class := Class{
		Name:           nameNode.Utf8Text(source),
		Package:        packageName,
		FilePath:       filePath,
		KDoc:           kdoc,
		Annotations:    anns,
		Imports:        imports,
		IsData:         flags["data"],
		IsEnum:         flags["enum"],
		IsSealed:       flags["sealed"],
		IsInterface:    flags["interface"] || node.Kind() == "interface_declaration" || directIndexOf(node, "interface") >= 0,
		TypeParameters: extractTypeParameters(node, source),
	}

	extractSuperTypes(node, source, &class)
	extractEnumEntries(node, source, &class)

	if primary := firstChildOfKind(node, "primary_constructor"); primary != nil {
		if classParams := firstChildOfKind(primary, "class_parameters"); classParams != nil {
			for i := uint(0); i < classParams.ChildCount(); i++ {
				param := classParams.Child(i)
				if param.Kind() != "class_parameter" {
					continue
				}
				prop, ok, propErr := extractClassParameter(param, source, filePath)
				if propErr != nil {
					return propErr
				}
				if ok {
					class.Fields = append(class.Fields, prop)
				}
			}
		}
	}

	if body := firstChildOfKind(node, "class_body"); body != nil {
		if err := extractClassBody(body, source, filePath, &class, result); err != nil {
			return err
		}
	}

	// An enum's entries live in enum_class_body, which sits beside class_body
	// on the declaration rather than inside it.
	if enumBody := firstChildOfKind(node, "enum_class_body"); enumBody != nil {
		if err := extractClassBody(enumBody, source, filePath, &class, result); err != nil {
			return err
		}
	}

	result.Classes = append(result.Classes, class)
	return nil
}

func extractClassBody(body *tree_sitter.Node, source []byte, filePath string, class *Class, result *ParseResult) error {
	for i := uint(0); i < body.ChildCount(); i++ {
		child := body.Child(i)
		switch child.Kind() {
		case "property_declaration":
			prop, err := extractProperty(child, source, filePath)
			if err != nil {
				return err
			}
			if prop != nil {
				class.Fields = append(class.Fields, *prop)
			}
		case "function_declaration":
			fn, err := extractFunction(child, source, filePath)
			if err != nil {
				return err
			}
			if fn != nil {
				class.Methods = append(class.Methods, *fn)
			}
		case "class_declaration", "object_declaration":
			if err := extractClass(child, source, class.Package, class.FilePath, class.Imports, result); err != nil {
				return err
			}
		case "enum_class_body":
			if err := extractClassBody(child, source, filePath, class, result); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractSuperTypes(node *tree_sitter.Node, source []byte, class *Class) {
	delegations := firstChildOfKind(node, "delegation_specifiers")
	if delegations == nil {
		return
	}
	for i := uint(0); i < delegations.ChildCount(); i++ {
		spec := delegations.Child(i)
		if spec.Kind() != "delegation_specifier" {
			continue
		}
		if invocation := firstChildOfKind(spec, "constructor_invocation"); invocation != nil {
			// The invocation text is "Base<A>()"; the type lives in its
			// user_type child. Reading the invocation directly would record
			// "Base<A>()" as the superclass name and lose inherited fields.
			userType := firstChildOfKind(invocation, "user_type")
			if userType == nil {
				continue
			}
			name, args := userTypeNameAndArgs(userType, source)
			if name == "" {
				continue
			}
			if class.SuperClass == "" {
				class.SuperClass = name
				class.SuperClassTypeArgs = args
			} else {
				class.Interfaces = append(class.Interfaces, name)
			}
			continue
		}
		if userType := firstChildOfKind(spec, "user_type"); userType != nil {
			name, _ := userTypeNameAndArgs(userType, source)
			if name != "" {
				class.Interfaces = append(class.Interfaces, name)
			}
		}
	}
}

// userTypeNameAndArgs splits "Base<A, B>" into "Base" and its arguments.
func userTypeNameAndArgs(node *tree_sitter.Node, source []byte) (string, []string) {
	raw := cleanTypeText(node.Utf8Text(source))
	if raw == "" {
		return "", nil
	}
	base, args := splitGenericType(raw)
	if args == nil {
		return base, nil
	}
	return base, args
}

func extractTypeParameters(node *tree_sitter.Node, source []byte) []string {
	params := firstChildOfKind(node, "type_parameters")
	if params == nil {
		return nil
	}
	var out []string
	for i := uint(0); i < params.ChildCount(); i++ {
		child := params.Child(i)
		if child.Kind() != "type_parameter" {
			continue
		}
		if id := firstChildOfKind(child, "identifier", "type_identifier"); id != nil {
			out = append(out, id.Utf8Text(source))
		}
	}
	return out
}

func extractEnumEntries(node *tree_sitter.Node, source []byte, class *Class) {
	body := firstChildOfKind(node, "enum_class_body")
	if body == nil {
		return
	}
	for i := uint(0); i < body.ChildCount(); i++ {
		child := body.Child(i)
		if child.Kind() != "enum_entry" {
			continue
		}
		if id := firstChildOfKind(child, "identifier"); id != nil {
			class.EnumValues = append(class.EnumValues, id.Utf8Text(source))
		}
	}
}

func extractClassParameter(node *tree_sitter.Node, source []byte, filePath string) (Property, bool, error) {
	modifiers := firstChildOfKind(node, "modifiers")
	anns, _, err := parseModifiers(modifiers, source, filePath)
	if err != nil {
		return Property{}, false, err
	}
	isProperty := directIndexOf(node, "val") >= 0 || directIndexOf(node, "var") >= 0
	if !isProperty {
		return Property{}, false, nil
	}
	nameNode := firstChildOfKind(node, "identifier")
	if nameNode == nil {
		return Property{}, false, nil
	}
	prop := Property{
		Name:        nameNode.Utf8Text(source),
		Type:        typeAfterColon(node, source),
		Annotations: anns,
		IsMutable:   directIndexOf(node, "var") >= 0,
		KDoc:        leadingKDoc(node, source),
	}
	if eq := directIndexOf(node, "="); eq >= 0 && eq+1 < int(node.ChildCount()) {
		prop.HasDefault = true
		prop.DefaultValue = strings.TrimSpace(node.Child(uint(eq) + 1).Utf8Text(source))
	}
	return prop, true, nil
}

func extractProperty(node *tree_sitter.Node, source []byte, filePath string) (*Property, error) {
	modifiers := firstChildOfKind(node, "modifiers")
	anns, flags, err := parseModifiers(modifiers, source, filePath)
	if err != nil {
		return nil, err
	}
	_ = flags
	decl := firstChildOfKind(node, "variable_declaration")
	if decl == nil {
		return nil, nil
	}
	nameNode := firstChildOfKind(decl, "identifier")
	if nameNode == nil {
		return nil, nil
	}
	prop := &Property{
		Name:        nameNode.Utf8Text(source),
		Type:        typeAfterColon(decl, source),
		Annotations: anns,
		IsMutable:   directIndexOf(node, "var") >= 0,
		KDoc:        leadingKDoc(node, source),
	}
	if prop.Type == "" {
		for i := uint(0); i < decl.ChildCount(); i++ {
			child := decl.Child(i)
			if isTypeNode(child.Kind()) {
				prop.Type = extractTypeNode(child, source)
				break
			}
		}
	}
	if eq := directIndexOf(node, "="); eq >= 0 && eq+1 < int(node.ChildCount()) {
		prop.HasDefault = true
		prop.DefaultValue = strings.TrimSpace(node.Child(uint(eq) + 1).Utf8Text(source))
	}
	return prop, nil
}

func extractFunction(node *tree_sitter.Node, source []byte, filePath string) (*Function, error) {
	modifiers := firstChildOfKind(node, "modifiers")
	anns, flags, err := parseModifiers(modifiers, source, filePath)
	if err != nil {
		return nil, err
	}
	nameNode := firstChildOfKind(node, "identifier")
	if nameNode == nil {
		return nil, nil
	}
	fn := &Function{
		Name:        nameNode.Utf8Text(source),
		Annotations: anns,
		KDoc:        leadingKDoc(node, source),
		IsOverride:  flags["override"],
	}

	if params := firstChildOfKind(node, "function_value_parameters"); params != nil {
		var pending []Annotation
		for i := uint(0); i < params.ChildCount(); i++ {
			child := params.Child(i)
			switch child.Kind() {
			case "parameter_modifiers", "modifiers":
				pendingAnns, _, modErr := parseModifiers(child, source, filePath)
				if modErr != nil {
					return nil, modErr
				}
				pending = append(pending, pendingAnns...)
			case "parameter":
				param, paramErr := extractParameter(child, source, pending)
				if paramErr != nil {
					return nil, paramErr
				}
				fn.Parameters = append(fn.Parameters, param)
				pending = nil
			}
		}
	}

	if returnType := typeAfterColon(node, source); returnType != "" {
		fn.ReturnType = returnType
		fn.HasDeclaredReturn = true
	} else if body := firstChildOfKind(node, "function_body"); body != nil {
		if strings.HasPrefix(strings.TrimSpace(body.Utf8Text(source)), "=") {
			// Expression body without a declared type. Kotlin infers it; a
			// source-only collector must not guess.
			fn.ReturnType = ""
		} else {
			fn.ReturnType = "Unit"
		}
	} else {
		fn.ReturnType = "Unit"
	}

	if body := firstChildOfKind(node, "function_body"); body != nil {
		fn.Body = body.Utf8Text(source)
	}
	return fn, nil
}

func extractParameter(node *tree_sitter.Node, source []byte, annotations []Annotation) (Parameter, error) {
	nameNode := firstChildOfKind(node, "identifier")
	if nameNode == nil {
		return Parameter{}, nil
	}
	param := Parameter{
		Name:        nameNode.Utf8Text(source),
		Type:        typeAfterColon(node, source),
		Annotations: annotations,
	}
	if eq := directIndexOf(node, "="); eq >= 0 && eq+1 < int(node.ChildCount()) {
		param.HasDefault = true
		param.DefaultValue = strings.TrimSpace(node.Child(uint(eq) + 1).Utf8Text(source))
	}
	return param, nil
}

func parseModifiers(node *tree_sitter.Node, source []byte, filePath string) ([]Annotation, map[string]bool, error) {
	flags := map[string]bool{}
	if node == nil {
		return nil, flags, nil
	}
	var anns []Annotation
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "annotation" {
			ann, err := extractAnnotation(child, nil, source, filePath)
			if err != nil {
				return nil, nil, err
			}
			anns = append(anns, ann)
			continue
		}
		collectModifierFlags(child, flags)
	}
	return anns, flags, nil
}

func collectModifierFlags(node *tree_sitter.Node, flags map[string]bool) {
	switch node.Kind() {
	case "data", "enum", "sealed", "open", "abstract", "final", "override", "suspend",
		"vararg", "value", "inner", "annotation", "fun", "interface", "private",
		"protected", "public", "internal":
		flags[node.Kind()] = true
	}
	if node.Kind() == "annotation" {
		return
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		collectModifierFlags(node.Child(i), flags)
	}
}

func leadingAnnotations(node *tree_sitter.Node, source []byte, filePath string) ([]Annotation, string, error) {
	var anns []Annotation
	var kdoc string
	for prev := node.PrevSibling(); prev != nil; prev = prev.PrevSibling() {
		switch prev.Kind() {
		case "line_comment":
			continue
		case "block_comment":
			text := strings.TrimSpace(prev.Utf8Text(source))
			if kdoc == "" && strings.HasPrefix(text, "/**") {
				kdoc = text
			}
			continue
		case "annotated_expression":
			got, err := collectAnnotatedExpression(prev, source, filePath)
			if err != nil {
				return nil, "", err
			}
			anns = append(got, anns...)
			continue
		default:
			if strings.TrimSpace(prev.Utf8Text(source)) == "" {
				continue
			}
			return anns, kdoc, nil
		}
	}
	return anns, kdoc, nil
}

func collectAnnotatedExpression(node *tree_sitter.Node, source []byte, filePath string) ([]Annotation, error) {
	var out []Annotation
	count := int(node.ChildCount())
	for i := 0; i < count; i++ {
		child := node.Child(uint(i))
		switch child.Kind() {
		case "annotation":
			var args *tree_sitter.Node
			if i+1 < count {
				next := node.Child(uint(i + 1))
				if next.Kind() == "parenthesized_expression" || next.Kind() == "value_arguments" {
					args = next
				}
			}
			ann, err := extractAnnotation(child, args, source, filePath)
			if err != nil {
				return nil, err
			}
			out = append(out, ann)
		case "annotated_expression":
			got, err := collectAnnotatedExpression(child, source, filePath)
			if err != nil {
				return nil, err
			}
			out = append(out, got...)
		}
	}
	return out, nil
}

func extractAnnotation(node *tree_sitter.Node, argsOverride *tree_sitter.Node, source []byte, filePath string) (Annotation, error) {
	ann := Annotation{Params: map[string]string{}}
	argsNode := argsOverride

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "use_site_target":
			ann.UseSiteTarget = strings.TrimSuffix(strings.TrimSpace(child.Utf8Text(source)), ":")
		case "user_type":
			ann.Name = lastSegment(cleanTypeText(child.Utf8Text(source)))
		case "constructor_invocation":
			if userType := firstChildOfKind(child, "user_type"); userType != nil {
				ann.Name = lastSegment(cleanTypeText(userType.Utf8Text(source)))
			}
			if valueArgs := firstChildOfKind(child, "value_arguments"); valueArgs != nil {
				argsNode = valueArgs
			}
		}
	}

	if ann.UseSiteTarget != "" {
		return Annotation{}, &UnsupportedConstructError{
			FilePath:  filePath,
			Construct: "@" + ann.UseSiteTarget + ":" + ann.Name,
		}
	}

	extractAnnotationParams(argsNode, source, ann.Params)
	return ann, nil
}

func extractAnnotationParams(node *tree_sitter.Node, source []byte, params map[string]string) {
	if node == nil {
		return
	}
	if node.Kind() == "parenthesized_expression" || node.Kind() == "value_arguments" {
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() == "value_argument" {
				readValueArgument(child, source, params)
				continue
			}
			if isAnnotationValueNode(child.Kind()) {
				setPositional(params, extractAnnotationValue(child, source))
			}
		}
		return
	}
}

func readValueArgument(node *tree_sitter.Node, source []byte, params map[string]string) {
	eq := directIndexOf(node, "=")
	if eq >= 0 {
		nameNode := firstChildOfKind(node, "identifier")
		if nameNode != nil {
			// Read the children after "=" only. Booleans such as
			// "required = false" arrive as identifier nodes, so skipping every
			// identifier would drop the value.
			var value string
			for i := uint(eq) + 1; i < node.ChildCount(); i++ {
				child := node.Child(i)
				if child.Kind() == "=" || child.Kind() == "," {
					continue
				}
				value = extractAnnotationValue(child, source)
				break
			}
			params[nameNode.Utf8Text(source)] = value
			return
		}
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" || child.Kind() == "=" || child.Kind() == "," {
			continue
		}
		if isAnnotationValueNode(child.Kind()) || child.Kind() == "collection_literal" || child.Kind() == "unary_expression" {
			setPositional(params, extractAnnotationValue(child, source))
			return
		}
	}
}

func setPositional(params map[string]string, value string) {
	if current, ok := params["value"]; ok && current != "" {
		params["value"] = current + "," + value
		return
	}
	params["value"] = value
}

func extractAnnotationValue(node *tree_sitter.Node, source []byte) string {
	switch node.Kind() {
	case "string_literal":
		return extractStringLiteral(node, source)
	case "collection_literal":
		var values []string
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() == "[" || child.Kind() == "]" || child.Kind() == "," {
				continue
			}
			values = append(values, extractAnnotationValue(child, source))
		}
		return strings.Join(values, ",")
	default:
		return strings.TrimSpace(node.Utf8Text(source))
	}
}

func extractStringLiteral(node *tree_sitter.Node, source []byte) string {
	text := strings.TrimSpace(node.Utf8Text(source))
	if strings.HasPrefix(text, "\"\"\"") && strings.HasSuffix(text, "\"\"\"") && len(text) >= 6 {
		return text[3 : len(text)-3]
	}
	if value, err := strconv.Unquote(text); err == nil {
		return value
	}
	if len(text) >= 2 && text[0] == '"' && text[len(text)-1] == '"' {
		return text[1 : len(text)-1]
	}
	return text
}

func isAnnotationValueNode(kind string) bool {
	switch kind {
	case "string_literal", "true", "false", "null", "integer_literal", "float_literal",
		"identifier", "navigation_expression", "user_type", "collection_literal",
		"unary_expression", "call_expression", "qualified_identifier":
		return true
	}
	return false
}

func leadingKDoc(node *tree_sitter.Node, source []byte) string {
	for prev := node.PrevSibling(); prev != nil; prev = prev.PrevSibling() {
		switch prev.Kind() {
		case "line_comment":
			continue
		case "block_comment":
			text := strings.TrimSpace(prev.Utf8Text(source))
			if strings.HasPrefix(text, "/**") {
				return text
			}
			return ""
		default:
			if strings.TrimSpace(prev.Utf8Text(source)) == "" {
				continue
			}
			return ""
		}
	}
	return ""
}

func mergeAnnotations(first, second []Annotation) []Annotation {
	out := make([]Annotation, 0, len(first)+len(second))
	seen := map[string]bool{}
	appendAnn := func(ann Annotation) {
		key := ann.Name + "|" + ann.Params["value"]
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, ann)
	}
	for _, ann := range first {
		appendAnn(ann)
	}
	for _, ann := range second {
		appendAnn(ann)
	}
	return out
}

func splitGenericType(raw string) (string, []string) {
	raw = strings.TrimSpace(raw)
	open := strings.Index(raw, "<")
	if open < 0 || !strings.HasSuffix(raw, ">") {
		return raw, nil
	}
	base := strings.TrimSpace(raw[:open])
	args := splitTopLevel(raw[open+1 : len(raw)-1])
	return base, args
}

func splitTopLevel(raw string) []string {
	var out []string
	depth := 0
	start := 0
	for i, r := range raw {
		switch r {
		case '<', '(', '[':
			depth++
		case '>', ')', ']':
			depth--
		case ',':
			if depth == 0 {
				out = append(out, strings.TrimSpace(raw[start:i]))
				start = i + 1
			}
		}
	}
	out = append(out, strings.TrimSpace(raw[start:]))
	return out
}

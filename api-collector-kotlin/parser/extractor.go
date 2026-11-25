package parser

import (
	"fmt"
	"strconv"
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// extractFile reads the declarations of one parsed file. With skipDamaged set,
// the tree holds syntax errors: a declaration they affect is skipped and named,
// with its line, in a warning instead of being read half-parsed.
func extractFile(root *tree_sitter.Node, source []byte, filePath string, result *ParseResult, skipDamaged bool) {
	packageName := extractPackageName(root, source)
	imports := extractImports(root, source)

	skip := func(node *tree_sitter.Node) {
		name := declaredName(node, source)
		if name == "" {
			name = "a declaration"
		}
		result.Warnings = append(result.Warnings, fmt.Sprintf("%s:%d: %s is skipped because the Kotlin grammar could not parse it",
			filePath, node.StartPosition().Row+1, name))
	}

	var visit func(node *tree_sitter.Node)
	visit = func(node *tree_sitter.Node) {
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			switch child.Kind() {
			case "ERROR":
				// The grammar folded a declaration into an error.
				if skipDamaged && !isStraySemicolon(child, source) {
					skip(child)
				}
			case "class_declaration", "object_declaration":
				if result.Error != nil {
					return
				}
				if skipDamaged && declarationDamaged(child, source) {
					skip(child)
					continue
				}
				if err := extractClass(child, source, packageName, filePath, imports, "", result); err != nil {
					result.Error = err
					return
				}
			case "property_declaration":
				if result.Error != nil {
					return
				}
				// Of the top-level properties only a const val can stand for
				// a string in an annotation.
				if !hasModifier(child, "const") {
					continue
				}
				if skipDamaged && declarationDamaged(child, source) {
					skip(child)
					continue
				}
				prop, err := extractProperty(child, source, filePath)
				if err != nil {
					result.Error = err
					return
				}
				if prop != nil {
					addFileConstant(result, packageName, filePath, imports, *prop)
				}
			case "annotated_expression":
				// A declaration can be wrapped in annotation expressions at
				// file scope. Descend so the class is not missed.
				visit(child)
			case "type_alias":
				if skipDamaged && declarationDamaged(child, source) {
					skip(child)
					continue
				}
				if alias, ok := extractTypeAlias(child, source, packageName, filePath, imports); ok {
					result.TypeAliases = append(result.TypeAliases, alias)
				}
			}
		}
	}
	visit(root)
}

// extractTypeAlias reads `typealias Name<T> = Target`. An alias whose name or
// target cannot be read is left out.
func extractTypeAlias(node *tree_sitter.Node, source []byte, packageName, filePath string, imports []Import) (TypeAlias, bool) {
	nameNode := node.ChildByFieldName("type")
	if nameNode == nil {
		return TypeAlias{}, false
	}
	var target *tree_sitter.Node
	sawEquals := false
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "=" {
			sawEquals = true
			continue
		}
		if sawEquals && isTypeNode(child.Kind()) {
			target = child
			break
		}
	}
	spelling := extractTypeNode(target, source)
	if spelling == "" {
		return TypeAlias{}, false
	}
	return TypeAlias{
		Name:           nameNode.Utf8Text(source),
		Package:        packageName,
		FilePath:       filePath,
		Target:         spelling,
		TargetNullable: typeNodeNullable(target),
		TypeParameters: extractTypeParameters(node, source),
		Imports:        imports,
		IsPrivate:      hasModifier(node, "private"),
	}, true
}

// addFileConstant records a top-level const val in the file facade of its
// file, creating the facade with the first one. A damaged file is extracted a
// declaration at a time, so the facade may already exist.
func addFileConstant(result *ParseResult, packageName, filePath string, imports []Import, prop Property) {
	for i := range result.Classes {
		if result.Classes[i].IsFileFacade && result.Classes[i].FilePath == filePath {
			result.Classes[i].Fields = append(result.Classes[i].Fields, prop)
			return
		}
	}
	result.Classes = append(result.Classes, Class{
		Name:         fileFacadeName(filePath),
		Package:      packageName,
		FilePath:     filePath,
		Imports:      imports,
		Fields:       []Property{prop},
		IsFileFacade: true,
	})
}

// fileFacadeName is the name of the class the JVM holds the top-level
// declarations of a file in: the file name, capitalised, followed by Kt.
// @file:JvmName is not read; Kotlin code never names the facade.
func fileFacadeName(filePath string) string {
	name := filePath
	if i := strings.LastIndexAny(name, `/\`); i >= 0 {
		name = name[i+1:]
	}
	if strings.HasSuffix(strings.ToLower(name), ".kt") {
		name = name[:len(name)-len(".kt")]
	}
	// A dependency source is named after the class it was fetched for.
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	if name == "" {
		return "Kt"
	}
	return strings.ToUpper(name[:1]) + name[1:] + "Kt"
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
		// A semicolon may end the import, as it ends a Java one.
		text = strings.TrimSpace(strings.TrimSuffix(text, ";"))
		alias := ""
		if idx := strings.Index(text, " as "); idx >= 0 {
			alias = strings.Trim(strings.TrimSpace(text[idx+len(" as "):]), "`")
			text = strings.TrimSpace(text[:idx])
		}
		if text == "" {
			continue
		}
		if strings.HasSuffix(text, ".*") {
			imports = append(imports, Import{Path: strings.TrimSuffix(text, ".*"), OnDemand: true})
			continue
		}
		imports = append(imports, Import{Path: text, Alias: alias})
	}
	return imports
}

// extractClass reads a class declaration, and the classes nested in it, into
// result. outer is the dotted path of the classes it is nested in, empty for a
// top-level class.
func extractClass(node *tree_sitter.Node, source []byte, packageName, filePath string, imports []Import, outer string, result *ParseResult) error {
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
		Outer:          outer,
		IsInner:        flags["inner"],
		IsPrivate:      flags["private"],
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
			nestedIn := class.Name
			if class.Outer != "" {
				nestedIn = class.Outer + "." + class.Name
			}
			if err := extractClass(child, source, class.Package, class.FilePath, class.Imports, nestedIn, result); err != nil {
				return err
			}
		case "companion_object":
			if err := extractCompanionConstants(child, source, filePath, class); err != nil {
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

// extractCompanionConstants reads the const vals of a companion object onto
// the class it belongs to. On the JVM they are static fields of that class,
// which is how Java and annotations reach them: Headers.VERSION. The rest of
// the companion is not instance state of the class and is not read.
func extractCompanionConstants(companion *tree_sitter.Node, source []byte, filePath string, class *Class) error {
	body := firstChildOfKind(companion, "class_body")
	if body == nil {
		return nil
	}
	for i := uint(0); i < body.ChildCount(); i++ {
		child := body.Child(i)
		if child.Kind() != "property_declaration" || !hasModifier(child, "const") {
			continue
		}
		prop, err := extractProperty(child, source, filePath)
		if err != nil {
			return err
		}
		if prop != nil {
			class.Fields = append(class.Fields, *prop)
		}
	}
	return nil
}

// hasModifier reports whether a declaration carries the modifier keyword.
func hasModifier(node *tree_sitter.Node, keyword string) bool {
	flags := map[string]bool{}
	if modifiers := firstChildOfKind(node, "modifiers"); modifiers != nil {
		collectModifierFlags(modifiers, flags)
	}
	return flags[keyword]
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
	raw := cleanTypeText(typeSource(node, source))
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
			class.EnumValueKDocs = append(class.EnumValueKDocs, leadingKDoc(child, source))
		}
	}
}

func extractClassParameter(node *tree_sitter.Node, source []byte, filePath string) (Property, bool, error) {
	modifiers := firstChildOfKind(node, "modifiers")
	anns, flags, err := parseModifiers(modifiers, source, filePath)
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
	if err := checkPropertyAnnotations(anns, nameNode.Utf8Text(source), filePath); err != nil {
		return Property{}, false, err
	}
	prop := Property{
		Name:        nameNode.Utf8Text(source),
		Type:        typeAfterColon(node, source),
		Annotations: anns,
		IsMutable:   directIndexOf(node, "var") >= 0,
		KDoc:        leadingKDoc(node, source),

		InConstructor: true,
	}
	// `vararg val ids: String` is declared with the type of one element; the
	// property holds an array of them, and the array itself is never null.
	if flags["vararg"] && prop.Type != "" {
		prop.Type = "Array<" + prop.Type + ">"
		prop.Vararg, prop.ElementNullable = true, nullableTypeAfterColon(node, source)
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
	decl := firstChildOfKind(node, "variable_declaration")
	if decl == nil {
		return nil, nil
	}
	nameNode := firstChildOfKind(decl, "identifier")
	if nameNode == nil {
		return nil, nil
	}
	if err := checkPropertyAnnotations(anns, nameNode.Utf8Text(source), filePath); err != nil {
		return nil, err
	}
	prop := &Property{
		Name:        nameNode.Utf8Text(source),
		Type:        typeAfterColon(decl, source),
		Annotations: anns,
		IsMutable:   directIndexOf(node, "var") >= 0,
		KDoc:        leadingKDoc(node, source),
		IsConst:     flags["const"],
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
		vararg := false
		for i := uint(0); i < params.ChildCount(); i++ {
			child := params.Child(i)
			switch child.Kind() {
			case "parameter_modifiers", "modifiers":
				pendingAnns, flags, modErr := parseModifiers(child, source, filePath)
				if modErr != nil {
					return nil, modErr
				}
				pending = append(pending, pendingAnns...)
				vararg = vararg || flags["vararg"]
			case "parameter":
				param, paramErr := extractParameter(child, source, pending)
				if paramErr != nil {
					return nil, paramErr
				}
				// `vararg ids: String` is declared with the type of one
				// element; the function receives an array of them, and the
				// array itself is never null.
				if vararg && param.Type != "" {
					param.Type = "Array<" + param.Type + ">"
					param.Vararg, param.ElementNullable = true, param.Nullable
					param.Nullable = false
				}
				vararg = false
				// The grammar puts a function parameter's default value
				// next to the parameter node, not inside it.
				if i+2 < params.ChildCount() && params.Child(i+1).Kind() == "=" {
					param.HasDefault = true
					param.DefaultValue = strings.TrimSpace(params.Child(i + 2).Utf8Text(source))
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
			fn.ReturnType = inferredUnit
		}
	} else {
		fn.ReturnType = inferredUnit
	}

	if body := firstChildOfKind(node, "function_body"); body != nil {
		fn.Body = body.Utf8Text(source)
	}
	return fn, nil
}

// inferredUnit is the return type of a function written without one and with a
// block body: Unit, by the language. It is the class of the standard library
// itself, written in full, since it was not written in the file and a class of
// the project that is called Unit has no part in it.
const inferredUnit = "kotlin.Unit"

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
	param.Nullable = nullableTypeAfterColon(node, source)
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
		"protected", "public", "internal", "const":
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

	// A use-site target (@field:, @get:, ...) is kept on the annotation. Whether
	// it can change the exported name is decided per property, where every
	// annotation of that property is in view; see checkPropertyAnnotations.
	extractAnnotationParams(argsNode, source, &ann)
	return ann, nil
}

// jacksonTargets are the use-site targets that land on a member Jackson reads
// for one logical property: the backing field, the getter, the setter, the
// setter parameter, or the constructor parameter. Jackson merges the
// annotations of these members, so an annotation on any of them applies to the
// whole property.
var jacksonTargets = map[string]bool{
	"":         true,
	"field":    true,
	"get":      true,
	"set":      true,
	"setparam": true,
	"param":    true,
}

// nameAffecting lists the annotations whose meaning is the property's exported
// name or presence. Every other annotation (validation, documentation,
// inclusion, JVM naming) leaves the name alone whatever its target.
var nameAffecting = map[string]bool{
	"JsonProperty": true,
	"JsonIgnore":   true,
}

// checkPropertyAnnotations decides whether the annotations of one property,
// use-site targets included, yield a single exported name.
//
// A target is only a problem when it can change that name. @field:NotNull or
// @get:JsonInclude is accepted as written. @get:JsonProperty("x") renames the
// property exactly as an untargeted @JsonProperty("x") does, because Jackson
// merges the field, getter, setter and constructor-parameter annotations of a
// property. What cannot be modelled fails loudly instead of exporting a guess:
// a name-affecting annotation on any use-site target Jackson does not read
// (anything outside jacksonTargets, e.g. @property:, @delegate:, @receiver:),
// names that disagree across targets, and @JsonIgnore together with an explicit
// name.
func checkPropertyAnnotations(anns []Annotation, propName, filePath string) error {
	var names []string
	var nameSources []string
	ignored := ""
	for _, ann := range anns {
		if !nameAffecting[ann.Name] {
			continue
		}
		construct := "@" + ann.Name
		if ann.UseSiteTarget != "" {
			construct = "@" + ann.UseSiteTarget + ":" + ann.Name
		}
		if !jacksonTargets[ann.UseSiteTarget] {
			return &UnsupportedConstructError{
				FilePath:  filePath,
				Construct: construct,
				Reason: "the " + ann.UseSiteTarget + " use-site target puts the annotation where Jackson does not read it, " +
					"so the exported name of property " + propName + " cannot be determined",
			}
		}
		switch ann.Name {
		case "JsonIgnore":
			if ann.Params["value"] != "false" {
				ignored = construct
			}
		case "JsonProperty":
			if name := ann.Params["value"]; name != "" {
				names = append(names, name)
				nameSources = append(nameSources, construct+"(\""+name+"\")")
			}
		}
	}
	for i := 1; i < len(names); i++ {
		if names[i] != names[0] {
			return &UnsupportedConstructError{
				FilePath:  filePath,
				Construct: strings.Join(nameSources, " "),
				Reason:    "property " + propName + " is given different names on different use-site targets",
			}
		}
	}
	if ignored != "" && len(names) > 0 {
		return &UnsupportedConstructError{
			FilePath:  filePath,
			Construct: ignored + " " + nameSources[0],
			Reason:    "property " + propName + " is both ignored and explicitly named, which Jackson splits by direction",
		}
	}
	return nil
}

func extractAnnotationParams(node *tree_sitter.Node, source []byte, ann *Annotation) {
	if node == nil {
		return
	}
	if node.Kind() == "parenthesized_expression" || node.Kind() == "value_arguments" {
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() == "value_argument" {
				readValueArgument(child, source, ann)
				continue
			}
			if isAnnotationValueNode(child.Kind()) {
				setPositional(ann, child, source)
			}
		}
		return
	}
}

func readValueArgument(node *tree_sitter.Node, source []byte, ann *Annotation) {
	eq := directIndexOf(node, "=")
	if eq >= 0 {
		nameNode := firstChildOfKind(node, "identifier")
		if nameNode != nil {
			// Read the children after "=" only. Booleans such as
			// "required = false" arrive as identifier nodes, so skipping every
			// identifier would drop the value.
			key := nameNode.Utf8Text(source)
			ann.Params[key] = ""
			for i := uint(eq) + 1; i < node.ChildCount(); i++ {
				child := node.Child(i)
				if child.Kind() == "=" || child.Kind() == "," {
					continue
				}
				setParam(ann, key, child, source)
				break
			}
			return
		}
	}
	// Without "=", the argument is positional and a lone identifier is its
	// value: a constant such as @RequestHeader(TENANT).
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "=" || child.Kind() == "," {
			continue
		}
		if isAnnotationValueNode(child.Kind()) || child.Kind() == "collection_literal" || child.Kind() == "unary_expression" {
			setPositional(ann, child, source)
			return
		}
	}
}

// setParam records one argument, marking it when it is written as a name.
func setParam(ann *Annotation, key string, value *tree_sitter.Node, source []byte) {
	ann.Params[key] = extractAnnotationValue(value, source)
	if isNameValue(value, source) {
		if ann.Refs == nil {
			ann.Refs = map[string]bool{}
		}
		ann.Refs[key] = true
	}
}

// setPositional records a positional argument as the value param. A second
// one joins the first, so the param is a list and no longer a single name.
func setPositional(ann *Annotation, value *tree_sitter.Node, source []byte) {
	if current, ok := ann.Params["value"]; ok && current != "" {
		ann.Params["value"] = current + "," + extractAnnotationValue(value, source)
		delete(ann.Refs, "value")
		return
	}
	setParam(ann, "value", value, source)
}

// isNameValue reports whether an annotation argument is written as a name,
// such as VERSION or Headers.VERSION, rather than as a literal. The grammar
// reads true, false and null as identifiers too.
func isNameValue(node *tree_sitter.Node, source []byte) bool {
	switch node.Kind() {
	case "identifier":
		switch strings.TrimSpace(node.Utf8Text(source)) {
		case "true", "false", "null":
			return false
		}
		return true
	case "navigation_expression", "qualified_identifier":
		return true
	}
	return false
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

package parser

import (
	"strings"

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

// extractImports collects the import declarations of a compilation unit.
//
// The declaration is read off its text rather than its child nodes: the
// grammar spells the trailing `.*` as an anonymous token, and `import static`
// as a keyword, so node kinds differ between grammar versions while the
// declaration text does not.
func extractImports(root *tree_sitter.Node, source []byte) []Import {
	var imports []Import

	for i := uint(0); i < root.ChildCount(); i++ {
		node := root.Child(i)
		if node.Kind() != "import_declaration" {
			continue
		}

		text := strings.TrimSpace(node.Utf8Text(source))
		text = strings.TrimSuffix(text, ";")
		text = strings.TrimSpace(strings.TrimPrefix(text, "import"))

		imp := Import{}
		if strings.HasPrefix(text, "static ") {
			imp.Static = true
			text = strings.TrimSpace(strings.TrimPrefix(text, "static"))
		}
		if strings.HasSuffix(text, ".*") {
			imp.OnDemand = true
			text = strings.TrimSuffix(text, ".*")
		}
		imp.Path = text
		if imp.Path == "" {
			continue
		}
		imports = append(imports, imp)
	}

	return imports
}

// enclosingTypes returns the dotted path of the types a declaration is nested
// in, outermost first: "Outer.Mid" for a class declared in Mid, which is
// declared in Outer. It is empty for a top-level declaration.
//
// A class declared in a method body is read as one of the types around the
// method: no other class can name it, and the type that holds the method is as
// close to where it belongs as the model goes.
func enclosingTypes(node *tree_sitter.Node, source []byte) string {
	var names []string
	for parent := node.Parent(); parent != nil; parent = parent.Parent() {
		switch parent.Kind() {
		case "class_declaration", "interface_declaration", "enum_declaration", "record_declaration", "annotation_type_declaration":
			if name := declaredName(parent, source); name != "" {
				names = append(names, name)
			}
		}
	}
	for i, j := 0, len(names)-1; i < j; i, j = i+1, j-1 {
		names[i], names[j] = names[j], names[i]
	}
	return strings.Join(names, ".")
}

// declaredName returns the name a type declaration gives itself.
func declaredName(node *tree_sitter.Node, source []byte) string {
	if name := node.ChildByFieldName("name"); name != nil {
		return name.Utf8Text(source)
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		if child := node.Child(i); child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

// qualifiedTypeText returns a qualified type name as written, without the
// white space the source may put between its parts and without the annotations
// inside it (see typeText): `Outer.Inner`.
func qualifiedTypeText(node *tree_sitter.Node, source []byte) string {
	return strings.Join(strings.Fields(typeText(node, source)), "")
}

// extractClass extracts class information including annotations and methods.
func extractClass(node *tree_sitter.Node, source []byte) Class {
	class := Class{}
	class.JavaDoc = extractLeadingJavaDoc(node, source)
	class.RawJavaDoc = rawLeadingJavaDoc(node, source)

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

	// A class that extends java.lang.Enum is an enum: the language gives no
	// other class that right. CFR prints a Kotlin enum this way, as a class
	// rather than an `enum` declaration, so a source decompiled from a JAR
	// without sources reaches the resolver in this form.
	class.IsEnum = extendsEnum(node, source)

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "class_body" {
			class.Methods = extractMethods(child, source)
			class.Fields = extractFields(child, source)
			if class.IsEnum {
				class.EnumConstants = decompiledEnumConstants(child, source)
				class.EnumConstantDocs = make([]string, len(class.EnumConstants))
			}
			break
		}
	}

	return class
}

// extendsEnum reports whether a class declaration names java.lang.Enum as its
// superclass, spelled `Enum<Self>` or `java.lang.Enum<Self>`.
func extendsEnum(node *tree_sitter.Node, source []byte) bool {
	for i := uint(0); i < node.ChildCount(); i++ {
		superclass := node.Child(i)
		if superclass.Kind() != "superclass" {
			continue
		}
		for j := uint(0); j < superclass.ChildCount(); j++ {
			typeNode := superclass.Child(j)
			if typeNode.Kind() == "generic_type" {
				// The base type is the first child, before the type arguments.
				if typeNode = typeNode.Child(0); typeNode == nil {
					return false
				}
			}
			switch typeNode.Kind() {
			case "type_identifier", "scoped_type_identifier":
				name := strings.Join(strings.Fields(typeNode.Utf8Text(source)), "")
				return name == "Enum" || name == "java.lang.Enum"
			}
		}
	}
	return false
}

// decompiledEnumConstants lists the constants of an enum that CFR printed as a
// class, in declaration order. CFR marks the field of each constant with an
// `/* enum */` comment after its modifiers:
//
//	public static final /* enum */ State PENDING = new State("PENDING");
//
// Only the marker tells a constant from another static field of the enum's own
// type, such as one a Kotlin companion object declares. The constants of a
// Kotlin enum are not always initialised at their declaration: CFR leaves the
// initializer out when the class initializer assigns them. The synthetic
// $VALUES and $ENTRIES carry no marker, so they are not constants.
func decompiledEnumConstants(classBody *tree_sitter.Node, source []byte) []string {
	var constants []string
	for i := uint(0); i < classBody.ChildCount(); i++ {
		field := classBody.Child(i)
		if field.Kind() != "field_declaration" || !hasEnumMarker(field, source) {
			continue
		}
		if declarator := field.ChildByFieldName("declarator"); declarator != nil {
			if name := declarator.ChildByFieldName("name"); name != nil {
				constants = append(constants, name.Utf8Text(source))
			}
		}
	}
	return constants
}

// hasEnumMarker reports whether a field declaration carries CFR's `/* enum */`
// comment.
func hasEnumMarker(field *tree_sitter.Node, source []byte) bool {
	for i := uint(0); i < field.ChildCount(); i++ {
		child := field.Child(i)
		if child.Kind() == "block_comment" && strings.TrimSpace(child.Utf8Text(source)) == "/* enum */" {
			return true
		}
	}
	return false
}

// extractInterface extracts interface information including annotations and methods.
func extractInterface(node *tree_sitter.Node, source []byte) Class {
	class := Class{IsInterface: true, JavaDoc: extractLeadingJavaDoc(node, source), RawJavaDoc: rawLeadingJavaDoc(node, source)}

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

	// An interface inherits the member types of the interfaces it extends.
	class.Interfaces = extractInterfaces(node, source)

	// Extract methods and constants from interface body
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "interface_body" {
			class.Methods = extractMethods(child, source)
			class.Fields = extractInterfaceConstants(child, source)
			break
		}
	}

	return class
}

// extractEnum reads an enum declaration: its name and its constants. The
// fields and methods of its body are not read: Spring binds a request
// parameter to an enum by the name of a constant, so the constants are the
// values it accepts.
func extractEnum(node *tree_sitter.Node, source []byte) Class {
	enum := Class{IsEnum: true, JavaDoc: extractLeadingJavaDoc(node, source), RawJavaDoc: rawLeadingJavaDoc(node, source)}
	enum.Annotations = extractAnnotations(node, source)
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch child.Kind() {
		case "identifier":
			if enum.Name == "" {
				enum.Name = child.Utf8Text(source)
			}
		case "enum_body":
			for j := uint(0); j < child.ChildCount(); j++ {
				constant := child.Child(j)
				if constant.Kind() != "enum_constant" {
					continue
				}
				if name := constant.ChildByFieldName("name"); name != nil {
					enum.EnumConstants = append(enum.EnumConstants, name.Utf8Text(source))
					enum.EnumConstantDocs = append(enum.EnumConstantDocs, extractLeadingJavaDoc(constant, source))
				}
			}
		}
	}
	return enum
}

// extractRecord extracts a record declaration. Its components become fields;
// the optional body (compact constructors, methods, static members) is ignored.
func extractRecord(node *tree_sitter.Node, source []byte) Class {
	rec := Class{JavaDoc: extractLeadingJavaDoc(node, source), RawJavaDoc: rawLeadingJavaDoc(node, source)}

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "identifier" {
			rec.Name = child.Utf8Text(source)
			break
		}
	}

	rec.Annotations = extractAnnotations(node, source)
	rec.TypeParameters = extractTypeParameters(node, source)
	rec.Interfaces = extractInterfaces(node, source)

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "formal_parameters" {
			rec.Fields = extractRecordComponents(child, source)
			break
		}
	}

	return rec
}

// extractRecordComponents converts formal_parameter record components to
// fields. Components are implicitly private final fields (JLS 8.10.3), so
// IsFinal is set; the resolver keeps instance final fields.
func extractRecordComponents(params *tree_sitter.Node, source []byte) []Field {
	var fields []Field

	for i := uint(0); i < params.ChildCount(); i++ {
		child := params.Child(i)
		if child.Kind() != "formal_parameter" && child.Kind() != "spread_parameter" {
			continue
		}
		param := extractParameter(child, source)
		if param.Name == "" || param.Type == "" {
			continue
		}
		fields = append(fields, Field{
			Name:        param.Name,
			Type:        param.Type,
			Annotations: param.Annotations,
			JavaDoc:     param.JavaDoc,
			IsFinal:     true,
		})
	}

	return fields
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
			extractAnnotationParams(child, source, &ann)
		}
	}

	return ann
}

// extractAnnotationParams extracts annotation parameters.
func extractAnnotationParams(node *tree_sitter.Node, source []byte, ann *Annotation) {
	set := func(key string, valueNode *tree_sitter.Node) {
		ann.Params[key] = extractAnnotationValue(valueNode, source)
		if isNameNode(valueNode.Kind()) {
			if ann.Refs == nil {
				ann.Refs = make(map[string]bool)
			}
			ann.Refs[key] = true
		}
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		if child.Kind() == "element_value_pair" {
			// The pair names its parts: the grammar labels the value, which
			// may itself be an identifier naming a constant.
			key := child.ChildByFieldName("key")
			value := child.ChildByFieldName("value")
			if key != nil && value != nil {
				set(key.Utf8Text(source), value)
			}
		} else if isAnnotationValueNode(child.Kind()) {
			set("value", child)
		}
	}
}

// isNameNode reports whether an annotation value is written as a name, such as
// VERSION or Headers.VERSION, rather than as a literal.
func isNameNode(kind string) bool {
	switch kind {
	case "identifier", "scoped_identifier", "field_access":
		return true
	}
	return false
}

// ExplicitAnnotationName returns the binding name carried by the named
// annotation, e.g. "@RequestHeader(\"Authorization\")" yields "Authorization".
// The single-value form and the explicit name entry are both accepted; an
// empty result means the annotation does not rename the bound element.
func ExplicitAnnotationName(annotations []Annotation, annName string) string {
	for _, ann := range annotations {
		if ann.Name != annName {
			continue
		}
		if v := strings.TrimSpace(ann.Params["value"]); v != "" {
			return v
		}
		return strings.TrimSpace(ann.Params["name"])
	}
	return ""
}

func extractAnnotationValue(node *tree_sitter.Node, source []byte) string {
	switch node.Kind() {
	case "string_literal":
		return extractStringLiteral(node, source)
	case "true", "false", "decimal_integer_literal", "identifier", "scoped_identifier", "field_access":
		return node.Utf8Text(source)
	case "element_value_array_initializer":
		var values []string
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if isAnnotationValueNode(child.Kind()) {
				values = append(values, extractAnnotationValue(child, source))
			}
		}
		return strings.Join(values, ",")
	default:
		return strings.TrimSpace(node.Utf8Text(source))
	}
}

func isAnnotationValueNode(kind string) bool {
	switch kind {
	case "string_literal", "true", "false", "decimal_integer_literal", "identifier", "scoped_identifier", "field_access", "element_value_array_initializer":
		return true
	default:
		return false
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

func extractLeadingJavaDoc(node *tree_sitter.Node, source []byte) string {
	return CleanDocComment(rawLeadingJavaDoc(node, source), false)
}

func rawLeadingJavaDoc(node *tree_sitter.Node, source []byte) string {
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

// CleanDocComment strips the markers of a Javadoc or KDoc comment and returns
// its text, one source line per line.
//
// Paragraphs stay apart: a blank line of the comment is kept as one blank
// line, so a reader or a Markdown renderer can tell a new paragraph from a
// wrapped line. Without tags, the text is the main description, which ends at
// the first block tag (@param, @return, @see, ...); a tag's continuation lines
// are not description either.
func CleanDocComment(raw string, includeTags bool) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	raw = strings.TrimPrefix(raw, "/**")
	raw = strings.TrimSuffix(raw, "*/")

	var lines []string
	blank := false
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line == "" {
			blank = len(lines) > 0
			continue
		}
		if !includeTags && strings.HasPrefix(line, "@") {
			break
		}
		if blank {
			lines = append(lines, "")
			blank = false
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func parseJavaDocParamsFromText(raw string) map[string]string {
	cleaned := CleanDocComment(raw, true)
	if cleaned == "" {
		return nil
	}

	params := make(map[string]string)
	for _, line := range strings.Split(cleaned, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "@param ") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "@param "))
		parts := strings.Fields(rest)
		if len(parts) == 0 {
			continue
		}
		name := parts[0]
		desc := strings.TrimSpace(strings.TrimPrefix(rest, name))
		params[name] = desc
	}
	if len(params) == 0 {
		return nil
	}
	return params
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

	method.JavaDoc = extractLeadingJavaDoc(node, source)
	method.RawJavaDoc = rawLeadingJavaDoc(node, source)
	method.JavaDocParams = parseJavaDocParamsFromText(method.RawJavaDoc)
	method.Annotations = extractAnnotations(node, source)

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "identifier":
			method.Name = child.Utf8Text(source)
		case "type_identifier", "generic_type", "integral_type", "floating_point_type", "boolean_type", "void_type",
			"array_type", "scoped_type_identifier":
			method.ReturnType = typeNodeText(child, source)
		case "formal_parameters":
			method.Parameters = extractParameters(child, source)
		case "modifiers":
			for j := uint(0); j < child.ChildCount(); j++ {
				if child.Child(j).Kind() == "static" {
					method.IsStatic = true
					break
				}
			}
		case "block":
			// The body is kept as source text: the frameworks that build the
			// response inside the method need to read it, and a node handle
			// would not outlive the parse tree.
			method.Body = child.Utf8Text(source)
		}
	}

	return method
}

// extractParameters extracts method parameters.
func extractParameters(node *tree_sitter.Node, source []byte) []Parameter {
	var params []Parameter

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "formal_parameter" || child.Kind() == "spread_parameter" {
			param := extractParameter(child, source)
			params = append(params, param)
		}
	}

	return params
}

// extractParameter extracts a single parameter. A varargs parameter
// (`String... ids`, a spread_parameter) is the array its type spells:
// `String[]`, which is what the method receives.
func extractParameter(node *tree_sitter.Node, source []byte) Parameter {
	param := Parameter{}
	dimensions := ""

	param.JavaDoc = extractLeadingJavaDoc(node, source)
	param.Annotations = extractAnnotations(node, source)

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "type_identifier", "generic_type", "integral_type", "floating_point_type", "boolean_type",
			"array_type", "scoped_type_identifier":
			param.Type = typeNodeText(child, source)
		case "identifier":
			param.Name = child.Utf8Text(source)
		case "variable_declarator":
			// A varargs parameter keeps its name in a declarator.
			if name := child.ChildByFieldName("name"); name != nil {
				param.Name = name.Utf8Text(source)
			}
		case "dimensions":
			// `String tags[]`: the brackets follow the name.
			dimensions = dimensionSuffix(child)
		}
	}
	param.Type += dimensions
	if node.Kind() == "spread_parameter" && param.Type != "" {
		param.Type += "[]"
	}

	return param
}

// typeNodeText spells a type node as written, except that an array type is
// always its element type followed by one `[]` per dimension: `String []`
// and `String[]` are the same type, and the resolver reads the brackets off
// the end of the spelling. A qualified name has no white space between its
// parts (see qualifiedTypeText).
func typeNodeText(node *tree_sitter.Node, source []byte) string {
	switch node.Kind() {
	case "array_type":
		element := node.ChildByFieldName("element")
		dimensions := node.ChildByFieldName("dimensions")
		if element != nil && dimensions != nil {
			return typeNodeText(element, source) + dimensionSuffix(dimensions)
		}
	case "scoped_type_identifier":
		return qualifiedTypeText(node, source)
	}
	return typeText(node, source)
}

// dimensionSuffix spells the dimensions node of an array type, or of the
// brackets written after a variable's name, as one `[]` per dimension. An
// annotation among the brackets (`String @A []`) is not part of the spelling.
func dimensionSuffix(dimensions *tree_sitter.Node) string {
	count := 0
	for i := uint(0); i < dimensions.ChildCount(); i++ {
		if dimensions.Child(i).Kind() == "[" {
			count++
		}
	}
	return strings.Repeat("[]", count)
}

// extractFields extracts all field declarations from a class body.
func extractFields(classBody *tree_sitter.Node, source []byte) []Field {
	var fields []Field

	for i := uint(0); i < classBody.ChildCount(); i++ {
		child := classBody.Child(i)
		if child.Kind() == "field_declaration" {
			fields = append(fields, extractFieldDeclaration(child, source, false)...)
		}
	}

	return fields
}

// extractInterfaceConstants extracts the constants an interface declares.
// Every field of an interface is implicitly public, static and final (JLS
// 9.3), so they are constants whether or not the modifiers are written.
func extractInterfaceConstants(body *tree_sitter.Node, source []byte) []Field {
	var fields []Field
	for i := uint(0); i < body.ChildCount(); i++ {
		child := body.Child(i)
		if child.Kind() != "constant_declaration" {
			continue
		}
		fields = append(fields, extractFieldDeclaration(child, source, true)...)
	}
	return fields
}

// extractFieldDeclaration extracts the fields one declaration of a class
// declares, or of an interface when inInterface is set, in the order they are
// written: `String a, b[] = {};` declares two. The fields share the
// modifiers, the annotations, the doc comment and the element type of the
// declaration; the brackets after a name and the initializer are each
// declarator's own.
func extractFieldDeclaration(node *tree_sitter.Node, source []byte, inInterface bool) []Field {
	template := Field{JavaDoc: extractLeadingJavaDoc(node, source), RawJavaDoc: rawLeadingJavaDoc(node, source)}
	template.IsStatic, template.IsFinal = inInterface, inInterface

	template.Annotations = extractAnnotations(node, source)

	var fieldType string
	var declarators []*tree_sitter.Node

	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)

		switch child.Kind() {
		case "modifiers":
			for j := uint(0); j < child.ChildCount(); j++ {
				modChild := child.Child(j)
				switch modChild.Kind() {
				case "static":
					template.IsStatic = true
				case "final":
					template.IsFinal = true
				}
			}
		case "type_identifier", "generic_type", "integral_type", "floating_point_type", "boolean_type",
			"array_type", "scoped_type_identifier":
			fieldType = typeNodeText(child, source)
		case "variable_declarator":
			declarators = append(declarators, child)
		}
	}

	if fieldType == "" {
		return nil
	}

	var fields []Field
	for _, declarator := range declarators {
		name := declarator.ChildByFieldName("name")
		if name == nil {
			continue
		}
		field := template
		field.Name = name.Utf8Text(source)
		field.Type = fieldType
		// `String tags[]`: the brackets follow the name.
		if brackets := declarator.ChildByFieldName("dimensions"); brackets != nil {
			field.Type += dimensionSuffix(brackets)
		}
		// The grammar carries the `= value` part as the declarator's
		// `value` field (an expression or array_initializer), so initializer
		// presence is read off that label rather than a node kind.
		value := declarator.ChildByFieldName("value")
		field.HasInitializer = value != nil
		if field.IsStatic && field.IsFinal && field.Type == "String" {
			field.Constant = constantInitializer(value, source)
		}
		fields = append(fields, field)
	}

	return fields
}

// constantInitializer returns the initializer of a String constant as written,
// when an annotation that names the constant can be given its value: a string
// literal, or the name of another constant. A text block and an expression
// such as a concatenation are left out.
func constantInitializer(value *tree_sitter.Node, source []byte) string {
	if value == nil {
		return ""
	}
	text := strings.TrimSpace(value.Utf8Text(source))
	switch {
	case value.Kind() == "string_literal" && !strings.HasPrefix(text, `"""`):
		return text
	case isNameNode(value.Kind()):
		return strings.Join(strings.Fields(text), "")
	}
	return ""
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
				case "scoped_type_identifier":
					return qualifiedTypeText(scChild, source), nil
				case "generic_type":
					return extractGenericBaseAndArgs(scChild, source)
				}
			}
		}
	}
	return "", nil
}

// extractInterfaces extracts interface names from a class declaration's implements
// clause, or from the extends clause of an interface declaration.
func extractInterfaces(node *tree_sitter.Node, source []byte) []string {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == "super_interfaces" || child.Kind() == "extends_interfaces" {
			var ifaces []string
			for j := uint(0); j < child.ChildCount(); j++ {
				ifaceChild := child.Child(j)
				switch ifaceChild.Kind() {
				case "type_identifier":
					ifaces = append(ifaces, ifaceChild.Utf8Text(source))
				case "scoped_type_identifier":
					ifaces = append(ifaces, qualifiedTypeText(ifaceChild, source))
				case "generic_type":
					baseName, _ := extractGenericBaseAndArgs(ifaceChild, source)
					ifaces = append(ifaces, baseName)
				case "type_list":
					for k := uint(0); k < ifaceChild.ChildCount(); k++ {
						listChild := ifaceChild.Child(k)
						switch listChild.Kind() {
						case "type_identifier":
							ifaces = append(ifaces, listChild.Utf8Text(source))
						case "scoped_type_identifier":
							ifaces = append(ifaces, qualifiedTypeText(listChild, source))
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
		case "scoped_type_identifier":
			baseName = qualifiedTypeText(child, source)
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
		case "type_identifier", "generic_type", "annotated_type", "array_type":
			args = append(args, typeText(child, source))
		case "scoped_type_identifier":
			args = append(args, qualifiedTypeText(child, source))
		}
	}

	return args
}

// typeText returns a type as written, except for two things.
//
// The annotations written inside it are left out: `List<@NotBlank(message = "x")
// String>` is `List<String>`. An annotation in a type argument is a type
// annotation. It says something of the elements, not of the declaration that
// holds the type, so it is no part of the type's name; kept in the spelling, it
// made `@NotBlank(message = X) String` a type of its own that no declaration
// could be found for. The annotation is left to the declaration's own
// annotations, which are read from its modifiers.
//
// An array type, wherever it stands, is its element type followed by one `[]`
// per dimension: `List<String []>` and `List<int [ ]>` are `List<String[]>` and
// `List<int[]>`. The resolver reads the brackets off the end of the spelling,
// so a space left in front of them or between them made the element a name
// that no declaration has.
func typeText(node *tree_sitter.Node, source []byte) string {
	var edits []typeEdit
	collectTypeEdits(node, &edits)
	if len(edits) == 0 {
		return node.Utf8Text(source)
	}

	start, end := node.StartByte(), node.EndByte()
	var out []byte
	pos := start
	for _, edit := range edits {
		out = append(out, source[pos:edit.start]...)
		out = append(out, edit.text...)
		pos = edit.end
		if edit.annotation {
			// The space that set the annotation apart from the type goes with it.
			for pos < end && isSpaceByte(source[pos]) {
				pos++
			}
		}
	}
	out = append(out, source[pos:end]...)
	return string(out)
}

// typeEdit is a range of the source text of a type and what stands in its place.
type typeEdit struct {
	start, end uint
	text       string
	annotation bool
}

// collectTypeEdits gathers the edits that spell the type under a node, in source
// order. An annotation is cut, and an array type has the space and the
// annotations in front of its brackets, and the brackets themselves, replaced by
// one `[]` per dimension. What lies inside either is not searched: an
// annotation's own arguments sit inside the range that is cut, and the
// annotations among the dimensions inside the range that is replaced.
func collectTypeEdits(node *tree_sitter.Node, edits *[]typeEdit) {
	switch node.Kind() {
	case "annotation", "marker_annotation":
		*edits = append(*edits, typeEdit{start: node.StartByte(), end: node.EndByte(), annotation: true})
		return
	case "array_type":
		element, dimensions := node.ChildByFieldName("element"), node.ChildByFieldName("dimensions")
		if element != nil && dimensions != nil {
			collectTypeEdits(element, edits)
			*edits = append(*edits, typeEdit{start: element.EndByte(), end: dimensions.EndByte(), text: dimensionSuffix(dimensions)})
			return
		}
	}
	for i := uint(0); i < node.ChildCount(); i++ {
		collectTypeEdits(node.Child(i), edits)
	}
}

func isSpaceByte(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
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

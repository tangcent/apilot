// Package resolver adapts Kotlin declarations to the Java type resolver so a
// structurally identical Kotlin source and Java source produce the same schema.
package resolver

import (
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	collector "github.com/tangcent/apilot/api-collector"
	javadoc "github.com/tangcent/apilot/api-collector-java/doc"
	javaparser "github.com/tangcent/apilot/api-collector-java/parser"
	javaresolver "github.com/tangcent/apilot/api-collector-java/resolver"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	model "github.com/tangcent/apilot/api-model"
)

// Resolver wraps the Java type resolver with Kotlin spelling and enum
// handling.
type Resolver struct {
	classes    []ktparser.Class
	byKey      map[string]*ktparser.Class
	bySimple   map[string][]string
	byFile     map[string][]string
	enumValues map[string][]string
	tr         *javaresolver.TypeResolver
	unresolved *collector.UnresolvedSet
	javaTypes  []javaparser.Class
	// scope binds the names the Kotlin files write (see typeScope).
	scope *typeScope
	// fileConstants maps the qualified name of a top-level const val
	// (com.example.VERSION) to the file facade that holds it.
	fileConstants map[string]*ktparser.Class
	// bindingTR resolves model attributes; it is built on first use.
	bindingTR *javaresolver.TypeResolver
	// The binding view is built on first use, so the settings it shares
	// with tr are kept here.
	dependencyResolver javaresolver.DependencyResolver
	requiredFields     string
}

// New builds a resolver over all Kotlin declarations in the project.
func New(classes []ktparser.Class) *Resolver {
	return NewWithAliases(classes, nil, nil)
}

// NewWithJavaTypes builds a resolver over the Kotlin declarations of a project
// that also holds Java sources. javaTypes are the Java declarations of the same
// tree: a Kotlin endpoint may take or return a class written in Java.
func NewWithJavaTypes(classes []ktparser.Class, javaTypes []javaparser.Class) *Resolver {
	return NewWithAliases(classes, javaTypes, nil)
}

// NewWithAliases builds a resolver over the declarations of a project, Kotlin
// classes and typealiases and, in a mixed tree, the Java classes. A name that
// reaches a typealias stands for the type the alias names.
func NewWithAliases(classes []ktparser.Class, javaTypes []javaparser.Class, aliases []ktparser.TypeAlias) *Resolver {
	scope := newTypeScope(classes, javaTypes)
	scope.addAliases(aliases)
	r := &Resolver{
		classes:    classes,
		byKey:      make(map[string]*ktparser.Class, len(classes)),
		bySimple:   make(map[string][]string),
		byFile:     make(map[string][]string),
		enumValues: make(map[string][]string),
		unresolved: collector.NewUnresolvedSet(),
		javaTypes:  javaTypes,
		scope:      scope,

		fileConstants: make(map[string]*ktparser.Class),
	}

	var javaClasses []javaparser.Class
	for i := range classes {
		class := &classes[i]
		key := qualifiedName(*class)
		if class.IsFileFacade {
			for _, prop := range class.Fields {
				r.fileConstants[qualify(class.Package, prop.Name)] = class
			}
		}
		r.byKey[key] = class
		r.bySimple[class.Name] = appendUnique(r.bySimple[class.Name], key)
		if class.FilePath != "" {
			r.byFile[class.FilePath] = appendUnique(r.byFile[class.FilePath], key)
		}
		if class.IsEnum {
			r.enumValues[class.Name] = class.EnumValues
			r.enumValues[key] = class.EnumValues
			// Enums are rendered as string options, not as empty objects.
			// The JSON view never sees the declaration; it is there for the
			// fields whose doc points at it with @see.
			javaClasses = append(javaClasses, toJavaEnum(*class))
			continue
		}
		javaClasses = append(javaClasses, r.scope.toJavaClass(*class))
	}
	javaClasses = append(javaClasses, javaTypes...)

	r.tr = javaresolver.NewTypeResolver(javaClasses)
	r.tr.SetUnresolved(r.unresolved)
	showStandardTypes(r.tr)
	return r
}

// showStandardTypes tells a resolver how to display the classes of the JDK that
// the adapter writes in full for the standard types of Kotlin: by their simple
// names, as a type that is shown or reported has always read.
func showStandardTypes(tr *javaresolver.TypeResolver) {
	for _, b := range builtinTypes {
		tr.ShowAs(b.name, lastSegment(b.name))
		if b.element != "" {
			tr.ShowAs(b.element, lastSegment(b.element))
		}
	}
}

// SetDependencyResolver lets the resolver expand types declared in the
// project's dependencies, such as a response envelope from a shared library.
// The binding view, built later, receives it too.
func (r *Resolver) SetDependencyResolver(dr javaresolver.DependencyResolver) {
	dr = projectEnumsFirst{dr: dr, r: r}
	r.dependencyResolver = dr
	r.tr.SetDependencyResolver(dr)
	if r.bindingTR != nil {
		r.bindingTR.SetDependencyResolver(dr)
	}
}

// projectEnumsFirst keeps the project's own enums away from the dependency
// resolver. The Java type table does not hold them, since the adapter renders
// them itself, so the Java resolver would otherwise ask the index for them and
// read some installed release of the project instead.
type projectEnumsFirst struct {
	dr javaresolver.DependencyResolver
	r  *Resolver
}

func (p projectEnumsFirst) ResolveClass(className string) *javaparser.Class {
	if c, ok := p.r.byKey[className]; ok && c.IsEnum {
		return nil
	}
	return p.dr.ResolveClass(className)
}

// Siblings passes on what the dependency resolver kept of the source it read a
// class from, for a resolver that keeps it: the classes nested in a class are
// read with it.
func (p projectEnumsFirst) Siblings(className string) []javaparser.Class {
	if reader, ok := p.dr.(interface {
		Siblings(className string) []javaparser.Class
	}); ok {
		return reader.Siblings(className)
	}
	return nil
}

// ResolveForBinding expands a type the way Spring's data binder sees it when
// it binds a model attribute from request parameters: properties go by their
// Kotlin names, because @JsonProperty and @JsonIgnore only shape JSON, and a
// val declared in the class body is left out, because nothing can set it.
func (r *Resolver) ResolveForBinding(source *ktparser.Class, rawType string, bindings map[string]string) *model.ObjectModel {
	if rawType == "" {
		return model.NullModel()
	}
	if r.bindingTR == nil {
		var javaClasses []javaparser.Class
		for _, class := range r.classes {
			if class.IsEnum {
				javaClasses = append(javaClasses, toJavaEnum(class))
			} else {
				javaClasses = append(javaClasses, r.scope.toBindingClass(class))
			}
		}
		javaClasses = append(javaClasses, r.javaTypes...)
		r.bindingTR = javaresolver.NewTypeResolver(javaClasses)
		r.bindingTR.SetUnresolved(r.unresolved)
		showStandardTypes(r.bindingTR)
		if r.dependencyResolver != nil {
			r.bindingTR.SetDependencyResolver(r.dependencyResolver)
		}
		if r.requiredFields != "" {
			r.bindingTR.SetRequiredFields(r.requiredFields)
		}
	}
	resolved := r.bindingTR.ResolveForBinding(r.scope.toJavaClass(*source), r.scope.normalize(source, rawType), bindingsOrNil(bindings))
	r.applyEnums(resolved)
	return resolved
}

// bindingsOrNil returns the bindings of type parameters, or nil when there is
// nothing to bind. They come from SuperClassBindings, which spells each
// argument in the Java names the shared resolver knows and writes it in full
// where it must (`java.lang.String`, `com.example.Item`), so they are passed on
// as they are: spelled again, a qualified name would be taken for another one.
func bindingsOrNil(bindings map[string]string) map[string]string {
	if len(bindings) == 0 {
		return nil
	}
	return bindings
}

// NormalizeType spells a type written in source the way the shared resolver
// reads it (see typeScope): what Kotlin binds to its standard library is the
// class of the JDK it stands for, written in full, and every other name is as
// written.
func (r *Resolver) NormalizeType(source *ktparser.Class, rawType string) string {
	if r == nil {
		return (*typeScope)(nil).normalize(source, rawType)
	}
	return r.scope.normalize(source, rawType)
}

// SuperClassBindings binds each type parameter of super to the argument that
// sub's extends clause gives it, spelled so that it means the same in the scope
// of super: `class Sub : Base<Item>()` binds Base's parameter to the class Item
// names where Sub is written, not to whichever class Item names beside Base.
// bindings binds the type parameters of sub itself, for an argument that
// mentions them.
func (r *Resolver) SuperClassBindings(sub, super *ktparser.Class, bindings map[string]string) map[string]string {
	return r.tr.SuperClassBindings(r.scope.toJavaClass(*sub), r.scope.toJavaClass(*super), bindingsOrNil(bindings))
}

// UnwrapOptional returns T of a type written `Optional<T>` in source when the
// name denotes java.util.Optional, and the type as written otherwise, both in
// the spelling the shared resolver reads (see NormalizeType). A class of the
// project called Optional is not looked through.
func (r *Resolver) UnwrapOptional(source *ktparser.Class, rawType string) string {
	javaSource := javaparser.Class{}
	if source != nil {
		javaSource = r.scope.toJavaClass(*source)
	}
	return r.tr.UnwrapOptional(javaSource, r.scope.normalize(source, rawType))
}

// javaClass converts the file a type is written in, which is the scope its
// names are bound in.
func (r *Resolver) javaClass(source *ktparser.Class) javaparser.Class {
	if source == nil {
		return javaparser.Class{}
	}
	return r.scope.toJavaClass(*source)
}

// Identity returns the qualified name that a class name written in source
// denotes, bound by the scope of its file (see
// javaresolver.TypeResolver.Identity). A name of the Kotlin standard library is
// the class of the JDK it stands for. Pass the base name of a type, not a type
// spelled with arguments.
func (r *Resolver) Identity(source *ktparser.Class, name string) string {
	return r.tr.Identity(r.javaClass(source), r.scope.normalize(source, name))
}

// IsOptionalType reports whether a type written in source is a java.util class
// that holds a value or nothing.
func (r *Resolver) IsOptionalType(source *ktparser.Class, rawType string) bool {
	return r.tr.IsOptionalType(r.javaClass(source), r.scope.normalize(source, rawType))
}

// IsMultipartFile reports whether a handler parameter of a type written in
// source is uploaded as parts of a multipart request: Spring's MultipartFile, or
// a collection or an array of them, held by a java.util.Optional or not. A
// class of the project that is called MultipartFile is not.
func (r *Resolver) IsMultipartFile(source *ktparser.Class, rawType string) bool {
	return r.tr.IsMultipartFile(r.javaClass(source), r.scope.normalize(source, rawType))
}

// IsUpload reports whether m is a model the resolver answered Spring's
// MultipartFile with, in the JSON view or the binding view. The model is shown
// as MultipartFile, as a class of the project that is called so is, so the
// model itself is what tells them apart.
func (r *Resolver) IsUpload(m *model.ObjectModel) bool {
	if r == nil {
		return false
	}
	return r.tr.IsUpload(m) || (r.bindingTR != nil && r.bindingTR.IsUpload(m))
}

// IsEnumIn reports whether a class name written in source denotes an enum of the
// project, by the scope of its file.
func (r *Resolver) IsEnumIn(source *ktparser.Class, name string) bool {
	if source == nil {
		return false
	}
	class, found := r.FindClass(source, strings.TrimSpace(name))
	return found && class.IsEnum
}

// SetRequiredFields selects how resolved fields are marked required: one of
// collector.RequiredFieldsDefault or collector.RequiredFieldsConstraints.
// The binding view, built later, uses the same mode.
func (r *Resolver) SetRequiredFields(mode string) {
	r.requiredFields = mode
	r.tr.SetRequiredFields(mode)
	if r.bindingTR != nil {
		r.bindingTR.SetRequiredFields(mode)
	}
}

// Unresolved returns type names the resolver could not expand, with enums
// filtered out: an enum is intentionally rendered as a string, so it is not a
// missing type.
func (r *Resolver) Unresolved() map[string]int {
	counts := r.unresolved.Counts()
	for name := range counts {
		if _, ok := r.enumValues[lastSegment(name)]; ok {
			delete(counts, name)
		}
	}
	return counts
}

// IsEnum reports whether name denotes an enum class.
func (r *Resolver) IsEnum(name string) bool {
	_, ok := r.enumValues[lastSegment(name)]
	return ok
}

// EnumValues returns the entries of an enum class.
func (r *Resolver) EnumValues(name string) []string {
	return r.enumValues[lastSegment(name)]
}

// ResolveFrom expands a Kotlin type spelling from the given source class.
func (r *Resolver) ResolveFrom(source *ktparser.Class, rawType string, bindings map[string]string) *model.ObjectModel {
	if rawType == "" {
		return model.NullModel()
	}
	resolved := r.tr.ResolveFrom(r.scope.toJavaClass(*source), r.scope.normalize(source, rawType), bindingsOrNil(bindings))
	r.applyEnums(resolved)
	return resolved
}

// maxConstantHops bounds a chain of top-level constants defined by one
// another, so a cycle cannot loop.
const maxConstantHops = 8

// ConstantValue returns the value of the String constant that ref names as
// written in source: VERSION, Headers.VERSION or com.example.Headers.VERSION.
//
// A const val of source, of its companion object or of a supertype, one of an
// object or a companion named by its class, and one imported by name
// (import com.example.Headers.VERSION) are static fields on the JVM and are
// found by the Java resolver, in the scanned sources or the dependencies. A
// top-level const val is a member of its package: it is found through an
// import of its name, in the package of source, through a star import, or by
// its qualified name.
func (r *Resolver) ConstantValue(source *ktparser.Class, ref string) (string, bool) {
	if source == nil {
		return "", false
	}
	return r.constantValue(source, strings.Join(strings.Fields(ref), ""), 0)
}

func (r *Resolver) constantValue(source *ktparser.Class, ref string, hops int) (string, bool) {
	if ref == "" || hops > maxConstantHops {
		return "", false
	}
	if value, ok := r.tr.ConstantValue(r.scope.toJavaClass(*source), ref); ok {
		return value, true
	}

	candidates := []string{ref}
	if !strings.Contains(ref, ".") {
		candidates = nil
		for _, imp := range source.Imports {
			if !imp.OnDemand && imp.Name() == ref {
				candidates = append(candidates, imp.Path)
			}
		}
		candidates = append(candidates, qualify(source.Package, ref))
		for _, imp := range source.Imports {
			if imp.OnDemand {
				candidates = append(candidates, imp.Path+"."+ref)
			}
		}
	}
	for _, key := range candidates {
		facade, found := r.fileConstants[key]
		if !found {
			continue
		}
		name := lastSegment(key)
		for _, prop := range facade.Fields {
			if prop.Name != name {
				continue
			}
			initializer := r.scope.constantInitializer(facade, prop)
			if initializer != "" && !strings.HasPrefix(initializer, `"`) {
				// The initializer names another constant, as written in the
				// file that declares this one.
				return r.constantValue(facade, initializer, hops+1)
			}
			return r.tr.ConstantValue(r.scope.toJavaClass(*facade), name)
		}
	}
	return "", false
}

func qualify(pkg, name string) string {
	if pkg == "" {
		return name
	}
	return pkg + "." + name
}

// FindClass resolves a Kotlin class in the same precedence order the Java
// resolver uses for simple names. A class nested in another is found from
// inside the class that declares it, or through it as Outer.Inner.
func (r *Resolver) FindClass(source *ktparser.Class, name string) (*ktparser.Class, bool) {
	return r.find(source, name, false)
}

// FindSupertype resolves a name that source writes in its supertype list. The
// list is outside the body of the class, where the classes nested in it are not
// in scope (the Base of `class Child : Base()` is the class the file imports,
// not a Child.Base), so the lookup starts at the class around source.
func (r *Resolver) FindSupertype(source *ktparser.Class, name string) (*ktparser.Class, bool) {
	return r.find(source, name, true)
}

// find is FindClass, or FindSupertype when header is set.
func (r *Resolver) find(source *ktparser.Class, name string, header bool) (*ktparser.Class, bool) {
	if name == "" {
		return nil, false
	}
	if strings.Contains(name, ".") {
		return r.findQualified(source, name, header)
	}
	if c, ok := r.findMember(source, name, header); ok {
		return c, true
	}
	for _, key := range r.byFile[source.FilePath] {
		if c := r.byKey[key]; c != nil && c.Name == name && c.Outer == "" {
			return c, true
		}
	}
	for _, imp := range source.Imports {
		if !imp.OnDemand && imp.Name() == name {
			if c, ok := r.byKey[imp.Path]; ok {
				return c, true
			}
		}
	}
	if key := source.Package + "." + name; source.Package != "" {
		if c, ok := r.byKey[key]; ok {
			return c, true
		}
	}
	for _, imp := range source.Imports {
		if imp.OnDemand {
			if c, ok := r.byKey[imp.Path+"."+name]; ok {
				return c, true
			}
		}
	}
	// A name nothing in scope declares falls back to a class that carries it, a
	// top-level one before any other. A class nested in another is named from
	// inside it or through it, so one of them answers only when it is the only
	// candidate: with two there is nothing to say which is meant.
	keys := append([]string(nil), r.bySimple[name]...)
	sort.Strings(keys)
	var nested *ktparser.Class
	candidates := 0
	for _, key := range keys {
		c := r.byKey[key]
		if c == nil {
			continue
		}
		if c.Outer == "" {
			return c, true
		}
		nested, candidates = c, candidates+1
	}
	return nested, candidates == 1
}

// findMember returns the class named name that is nested in source or in a
// class around it, the innermost first. Those of source itself are left out when
// header is set.
func (r *Resolver) findMember(source *ktparser.Class, name string, header bool) (*ktparser.Class, bool) {
	for c := source; c != nil; {
		if !header || c != source {
			if member, ok := r.byKey[qualifiedName(*c)+"."+name]; ok {
				return member, true
			}
		}
		if c.Outer == "" {
			return nil, false
		}
		c = r.byKey[qualify(c.Package, c.Outer)]
	}
	return nil, false
}

// findQualified resolves a dotted name: a class named through the classes it is
// nested in (Outer.Inner), or by its qualified name. The first part names a
// class of its own or it does not; in the second case the whole name is the key
// of a class.
func (r *Resolver) findQualified(source *ktparser.Class, name string, header bool) (*ktparser.Class, bool) {
	head, rest, _ := strings.Cut(name, ".")
	if first, _ := utf8.DecodeRuneInString(head); head != "" && !unicode.IsLower(first) {
		if class, ok := r.find(source, head, header); ok {
			key := qualifiedName(*class)
			for _, segment := range strings.Split(rest, ".") {
				key += "." + segment
			}
			if c, ok := r.byKey[key]; ok {
				return c, true
			}
		}
	}
	c, ok := r.byKey[name]
	return c, ok
}

func (r *Resolver) applyEnums(m *model.ObjectModel) {
	if m == nil {
		return
	}
	switch m.Kind {
	case model.KindSingle:
		if r.IsEnum(m.TypeName) {
			m.TypeName = model.JsonTypeString
		}
	case model.KindObject:
		for _, field := range m.Fields {
			if field == nil || field.Model == nil {
				continue
			}
			if field.Model.Kind == model.KindSingle {
				if options := r.enumOptions(field.Model.TypeName); len(options) > 0 {
					field.Model = model.SingleModel(model.JsonTypeString)
					field.Options = options
					continue
				}
			}
			r.applyEnums(field.Model)
		}
	case model.KindArray:
		r.applyEnums(m.Items)
	case model.KindMap:
		r.applyEnums(m.KeyModel)
		r.applyEnums(m.ValueModel)
	}
}

// enumOptions returns the values of the enum a type name denotes, by its
// simple name as EnumValues goes, each described by the KDoc of its entry.
func (r *Resolver) enumOptions(name string) []model.FieldOption {
	values := r.EnumValues(name)
	if len(values) == 0 {
		return nil
	}
	for _, key := range r.bySimple[lastSegment(name)] {
		if c := r.byKey[key]; c != nil && c.IsEnum && slices.Equal(c.EnumValues, values) {
			return javaresolver.EnumOptions(toJavaEnum(*c))
		}
	}
	options := make([]model.FieldOption, 0, len(values))
	for _, value := range values {
		options = append(options, model.FieldOption{Value: value})
	}
	return options
}

// ToJavaClasses converts Kotlin declarations to the Java model, so a Java
// endpoint in the same tree can resolve them. An enum keeps its constants for
// the binding view; the Java resolver's JSON view does not see enums.
//
// javaTypes are the Java declarations of the same tree and aliases the Kotlin
// typealiases: a name in a Kotlin file binds to them as it does in the resolver
// built over the same declarations.
func ToJavaClasses(classes []ktparser.Class, javaTypes []javaparser.Class, aliases []ktparser.TypeAlias) []javaparser.Class {
	scope := newTypeScope(classes, javaTypes)
	scope.addAliases(aliases)
	out := make([]javaparser.Class, 0, len(classes))
	for _, class := range classes {
		if class.IsEnum {
			out = append(out, toJavaEnum(class))
			continue
		}
		out = append(out, scope.toJavaClass(class))
	}
	return out
}

// toJavaEnum converts an enum to the Java model: its entries, each with its
// doc comment cleaned.
func toJavaEnum(class ktparser.Class) javaparser.Class {
	out := javaparser.Class{
		Name:          class.Name,
		Package:       class.Package,
		Outer:         class.Outer,
		FilePath:      class.FilePath,
		Imports:       toJavaImports(class.Imports),
		IsEnum:        true,
		EnumConstants: append([]string(nil), class.EnumValues...),
	}
	for i := range class.EnumValues {
		doc := ""
		if i < len(class.EnumValueKDocs) {
			doc = javadoc.CleanJavaDoc(class.EnumValueKDocs[i])
		}
		out.EnumConstantDocs = append(out.EnumConstantDocs, doc)
	}
	return out
}

// toJavaImports converts the imports of a file. A renaming import binds a name
// no compilation unit of the Java model knows, and its uses are written by the
// class's own name (see typeScope), so it is left out: the class's own simple
// name is not bound by it.
func toJavaImports(imports []ktparser.Import) []javaparser.Import {
	out := make([]javaparser.Import, 0, len(imports))
	for _, imp := range imports {
		if imp.Alias != "" {
			continue
		}
		out = append(out, javaparser.Import{Path: imp.Path, OnDemand: imp.OnDemand})
	}
	return out
}

// toBindingClass converts a declaration for the binding view: every property
// Spring can set, under its Kotlin name.
func (s *typeScope) toBindingClass(class ktparser.Class) javaparser.Class {
	out := s.toJavaClass(ktparser.Class{
		Name: class.Name, Package: class.Package, Outer: class.Outer, IsInner: class.IsInner, FilePath: class.FilePath, KDoc: class.KDoc,
		Annotations: class.Annotations, TypeParameters: class.TypeParameters,
		SuperClass: class.SuperClass, SuperClassTypeArgs: class.SuperClassTypeArgs,
		Interfaces: class.Interfaces, Imports: class.Imports,
		IsInterface: class.IsInterface, IsSealed: class.IsSealed,
	})
	for _, prop := range class.Fields {
		if !prop.IsMutable && !prop.InConstructor {
			continue
		}
		out.Fields = append(out.Fields, javaparser.Field{
			Name:        prop.Name,
			Type:        s.normalize(&class, prop.Type),
			Annotations: fieldAnnotations(prop.Annotations),
			JavaDoc:     javadoc.CleanJavaDoc(prop.KDoc),
			IsFinal:     !prop.IsMutable,
			RawJavaDoc:  prop.KDoc,
		})
	}
	return out
}

func (s *typeScope) toJavaClass(class ktparser.Class) javaparser.Class {
	// The Java parser cleans a doc comment when it reads it; the Kotlin parser
	// keeps the raw "/** ... */" text, so it is cleaned here, where the
	// declaration enters the shared resolver.
	out := javaparser.Class{
		Name:           class.Name,
		Package:        class.Package,
		Outer:          class.Outer,
		Inner:          class.IsInner,
		FilePath:       class.FilePath,
		JavaDoc:        javadoc.CleanJavaDoc(class.KDoc),
		Annotations:    toJavaAnnotations(class.Annotations),
		TypeParameters: append([]string(nil), class.TypeParameters...),
		SuperClass:     s.normalizeHeader(&class, class.SuperClass),
		Interfaces:     make([]string, 0, len(class.Interfaces)),
		IsInterface:    class.IsInterface || class.IsSealed,
	}
	// The supertype list is outside the body of the class.
	for _, arg := range class.SuperClassTypeArgs {
		out.SuperClassTypeArgs = append(out.SuperClassTypeArgs, s.normalizeHeader(&class, arg))
	}
	for _, iface := range class.Interfaces {
		out.Interfaces = append(out.Interfaces, s.normalizeHeader(&class, iface))
	}
	out.Imports = toJavaImports(class.Imports)
	for _, prop := range class.Fields {
		// An @JsonIgnore(false) re-enables the property; only an ignore whose
		// value is not "false" drops the field, matching the parser's validation.
		if hasAnnotation(prop.Annotations, "JsonIgnore") && annotationValue(prop.Annotations, "JsonIgnore") != "false" {
			continue
		}
		if prop.IsConst {
			// A const val is a static field on the JVM: no JSON property, but
			// a constant an annotation can name.
			out.Fields = append(out.Fields, javaparser.Field{
				Name:           prop.Name,
				Type:           s.normalize(&class, prop.Type),
				IsStatic:       true,
				IsFinal:        true,
				HasInitializer: prop.HasDefault,
				Constant:       s.constantInitializer(&class, prop),
			})
			continue
		}
		name := prop.Name
		if jsonName := propertyName(prop.Annotations); jsonName != "" {
			name = jsonName
		}
		out.Fields = append(out.Fields, javaparser.Field{
			Name:        name,
			Type:        s.normalize(&class, prop.Type),
			Annotations: fieldAnnotations(prop.Annotations),
			JavaDoc:     javadoc.CleanJavaDoc(prop.KDoc),
			IsFinal:     !prop.IsMutable,
			RawJavaDoc:  prop.KDoc,
		})
	}
	// Methods are deliberately not copied: Kotlin data-class getters,
	// componentN, copy and toString are compiler-generated. Properties are the
	// only JSON fields.
	return out
}

func toJavaAnnotations(annotations []ktparser.Annotation) []javaparser.Annotation {
	if len(annotations) == 0 {
		return nil
	}
	out := make([]javaparser.Annotation, 0, len(annotations))
	for _, ann := range annotations {
		out = append(out, javaparser.Annotation{Name: ann.Name, Params: ann.Params, Refs: ann.Refs})
	}
	return out
}

// constantInitializer returns the initializer of a String const val the way
// the Java model keeps a constant's: a string literal, quoted, or the name of
// another constant. A string template, a raw string spanning lines, and any
// other expression are left out: their value cannot be read off the source.
func (s *typeScope) constantInitializer(unit *ktparser.Class, prop ktparser.Property) string {
	switch s.normalize(unit, prop.Type) {
	case "", "java.lang.String":
	default:
		return ""
	}
	value := strings.TrimSpace(prop.DefaultValue)
	switch {
	case strings.HasPrefix(value, `"""`):
		if len(value) < 6 || !strings.HasSuffix(value, `"""`) || strings.ContainsAny(value, "$\n") {
			return ""
		}
		return strconv.Quote(value[3 : len(value)-3])
	case strings.HasPrefix(value, `"`):
		if hasTemplate(value) {
			return ""
		}
		return value
	case isQualifiedName(value):
		return value
	}
	return ""
}

// hasTemplate reports whether a Kotlin string literal interpolates a value:
// a $ before a name or a brace that no backslash escapes.
func hasTemplate(literal string) bool {
	for i := 0; i < len(literal)-1; i++ {
		switch literal[i] {
		case '\\':
			i++
		case '$':
			if next := literal[i+1]; next == '{' || next == '_' || unicode.IsLetter(rune(next)) {
				return true
			}
		}
	}
	return false
}

// isQualifiedName reports whether text is a name such as VERSION or
// Headers.VERSION.
func isQualifiedName(text string) bool {
	if text == "" {
		return false
	}
	for _, part := range strings.Split(text, ".") {
		if part == "" {
			return false
		}
		for i, r := range part {
			if r != '_' && !unicode.IsLetter(r) && (i == 0 || !unicode.IsDigit(r)) {
				return false
			}
		}
	}
	return true
}

func fieldAnnotations(annotations []ktparser.Annotation) []javaparser.Annotation {
	out := toJavaAnnotations(annotations)
	for _, ann := range annotations {
		if ann.Name == "JsonProperty" && ann.Params["required"] == "false" {
			out = append(out, javaparser.Annotation{Name: "Nullable"})
		}
	}
	return out
}

func propertyName(annotations []ktparser.Annotation) string {
	for _, ann := range annotations {
		if ann.Name == "JsonProperty" {
			if name := ann.Params["value"]; name != "" {
				return name
			}
		}
	}
	return ""
}

func hasAnnotation(annotations []ktparser.Annotation, name string) bool {
	for _, ann := range annotations {
		if ann.Name == name {
			return true
		}
	}
	return false
}

func annotationValue(annotations []ktparser.Annotation, name string) string {
	base := lastSegment(name)
	param := ""
	if idx := strings.Index(name, "."); idx >= 0 {
		param = name[idx+1:]
	}
	for _, ann := range annotations {
		if ann.Name != base {
			continue
		}
		if param == "" {
			if v, ok := ann.Params["value"]; ok {
				return v
			}
			return "true"
		}
		if v, ok := ann.Params[param]; ok {
			return v
		}
	}
	return ""
}

// qualifiedName returns the key of a declaration: its package, the classes it is
// nested in and its name, as the Java resolver registers it.
func qualifiedName(class ktparser.Class) string {
	name := class.Name
	if class.Outer != "" {
		name = class.Outer + "." + name
	}
	return qualify(class.Package, name)
}

func lastSegment(name string) string {
	if idx := strings.LastIndex(name, "."); idx >= 0 {
		return name[idx+1:]
	}
	return name
}

func appendUnique(list []string, value string) []string {
	for _, item := range list {
		if item == value {
			return list
		}
	}
	return append(list, value)
}

func splitGenericType(raw string) (string, []string) {
	raw = strings.TrimSpace(raw)
	open := strings.Index(raw, "<")
	if open < 0 || !strings.HasSuffix(raw, ">") {
		return raw, nil
	}
	base := strings.TrimSpace(raw[:open])
	return base, splitTopLevel(raw[open+1 : len(raw)-1])
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

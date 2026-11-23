package springmvc

import (
	"sort"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/resolver"
)

// A handler parameter is a framework object, a simple value, a collection, a
// Pageable, a Sort or a file by the class its type names, and the scope of the
// handler's file says which class that is. A class or an enum of the project that
// bears the name of one of them is another class, wherever it sits: in the
// package of the handler, imported by name, imported on demand. The tests below
// give every name of the tables such a class and ask how a parameter of that
// type is bound, and ask the same of the library class reached each way a file
// can reach it.

// classifyCase is a type name of the tables, how a parameter that spells it is
// bound when it is the library's class, and how it is spelled.
type classifyCase struct {
	name     string
	spelling string
	want     ImplicitKind
}

func classifyCases() []classifyCase {
	var cases []classifyCase
	for name := range frameworkTypes {
		spelling := name
		if name == "Map" || name == "MultiValueMap" {
			spelling += "<String, String>"
		}
		cases = append(cases, classifyCase{name, spelling, ImplicitNone})
	}
	for name := range simpleValueTypes {
		cases = append(cases, classifyCase{name, name, ImplicitParam})
	}
	for name := range collectionTypes {
		cases = append(cases, classifyCase{name, name + "<String>", ImplicitParam})
	}
	cases = append(cases,
		classifyCase{"Pageable", "Pageable", ImplicitPageable},
		classifyCase{"Sort", "Sort", ImplicitSort},
		classifyCase{"MultipartFile", "MultipartFile", ImplicitFile},
		classifyCase{"MultipartFile", "List<MultipartFile>", ImplicitFile},
		classifyCase{"MultipartFile", "MultipartFile[]", ImplicitFile},
	)
	sort.Slice(cases, func(i, j int) bool { return cases[i].spelling < cases[j].spelling })
	return cases
}

const (
	handlerPackage = "com.example.handler"
	ownPackage     = "com.example.own"
	otherPackage   = "com.example.other"
)

// A name of the tables is the library's class in a handler that imports it, by
// name or on demand, or writes it in full.
func TestClassifyImplicit_ALibraryClassIsOneByItsIdentity(t *testing.T) {
	for _, tc := range classifyCases() {
		for _, pkg := range resolver.LibraryPackages(tc.name) {
			qualified := pkg + "." + tc.name
			collection := resolver.LibraryPackages("List")[0] + ".List"
			handlers := map[string]handlerScope{
				"single-type import": newHandlerScope(handlerPackage, importing(qualified, collection),
					projectClass(otherPackage, tc.name)),
				"import on demand": newHandlerScope(handlerPackage, []parser.Import{
					{Path: pkg, OnDemand: true}, {Path: "java.util", OnDemand: true},
				}, projectClass(otherPackage, tc.name)),
				// A class of another package that nothing imports is not in scope.
				"project class of another package": newHandlerScope(handlerPackage, importing(qualified, collection),
					projectClass(ownPackage, tc.name), projectEnum(otherPackage, tc.name)),
			}
			for scope, handler := range handlers {
				if got, _ := handler.classify(tc.spelling); got != tc.want {
					t.Errorf("%s of %s, %s: %s = %v, want %v", tc.name, pkg, scope, tc.spelling, got, tc.want)
				}
			}
			// Written in full, the name needs no import and no class of the
			// project can take its place.
			if tc.spelling == tc.name {
				handler := newHandlerScope(handlerPackage, nil, projectClass(handlerPackage, tc.name))
				if got, _ := handler.classify(qualified); got != tc.want {
					t.Errorf("%s written in full = %v, want %v", qualified, got, tc.want)
				}
			}
		}
	}
}

// A class of the project that bears such a name is the class in its own scope:
// a bean is bound property by property, and an enum is a request parameter,
// whatever the table says about the name.
func TestClassifyImplicit_AProjectClassOfThatNameIsNotOne(t *testing.T) {
	for _, tc := range classifyCases() {
		if tc.spelling != tc.name && tc.name != "Map" && tc.name != "MultiValueMap" {
			// The element decides the rest of the spellings; the bare name is
			// what the class takes.
			continue
		}
		spelling := tc.name
		if tc.name == "Map" || tc.name == "MultiValueMap" {
			spelling = tc.spelling
		}
		pkg := resolver.LibraryPackages(tc.name)[0]
		for scope, handler := range map[string]func(parser.Class) handlerScope{
			"same package": func(c parser.Class) handlerScope {
				c.Package = handlerPackage
				return newHandlerScope(handlerPackage, nil, c)
			},
			"single-type import": func(c parser.Class) handlerScope {
				return newHandlerScope(handlerPackage, importing(ownPackage+"."+tc.name), c)
			},
			"import on demand": func(c parser.Class) handlerScope {
				return newHandlerScope(handlerPackage, []parser.Import{{Path: ownPackage, OnDemand: true}}, c)
			},
			// The class of the package wins over an import on demand of the
			// library's package, which only supplies what the package does not
			// declare.
			"same package beats the library on demand": func(c parser.Class) handlerScope {
				c.Package = handlerPackage
				return newHandlerScope(handlerPackage, []parser.Import{{Path: pkg, OnDemand: true}}, c)
			},
		} {
			bean := handler(projectClass(ownPackage, tc.name))
			if got, _ := bean.classify(spelling); got != ImplicitModel {
				t.Errorf("a class %s, %s: %s = %v, want a model attribute", tc.name, scope, spelling, got)
			}
			enum := handler(projectEnum(ownPackage, tc.name))
			if got, _ := enum.classify(spelling); got != ImplicitParam {
				t.Errorf("an enum %s, %s: %s = %v, want a request parameter", tc.name, scope, spelling, got)
			}
		}
	}
}

// An import of another library's class of that name, by name, is that class.
func TestClassifyImplicit_AnotherLibrarysClassOfThatNameIsNotOne(t *testing.T) {
	for _, tc := range classifyCases() {
		if tc.spelling != tc.name {
			continue
		}
		handler := newHandlerScope(handlerPackage, importing("org.example.legacy."+tc.name))
		if got, _ := handler.classify(tc.name); got != ImplicitModel {
			t.Errorf("org.example.legacy.%s = %v, want a model attribute", tc.name, got)
		}
	}
}

// The names of the tables are in the table of the classes the resolver knows by
// their packages, so that no name is left to be read by its simple name.
func TestClassifyImplicit_EveryNameOfTheTablesHasItsPackages(t *testing.T) {
	for _, table := range []map[string]bool{frameworkTypes, simpleValueTypes, collectionTypes} {
		for name := range table {
			if len(resolver.LibraryPackages(name)) == 0 {
				t.Errorf("%s is in a table of the classifier but no package is known to declare it", name)
			}
		}
	}
	for _, name := range []string{"Pageable", "Sort", "MultipartFile", "Optional", "OptionalInt", "OptionalLong", "OptionalDouble"} {
		if len(resolver.LibraryPackages(name)) == 0 {
			t.Errorf("%s has no package", name)
		}
	}
}

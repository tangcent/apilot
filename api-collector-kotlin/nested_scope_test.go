package kotlincollector

import (
	"path/filepath"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// nestedScopeExpectation is the export of testdata/nested-scope. A class nested
// in another is in scope around its outer class only, and a private class in its
// own file only: everywhere else Int, Number and Long are Kotlin's, and a
// function written without a return type returns Unit, whatever else the
// package declares under that name.
var nestedScopeExpectation = map[string]map[string]string{
	"/nest/save": {
		"request.count": "int", "request.weight": "double", "request.counts": "array<int>", "request.total": "long",
		"response.count": "int", "response.weight": "double", "response.counts": "array<int>", "response.total": "long",
	},
	"/nest/count": {"response": "int"},
	"/nest/clear": {"response": "nil"},
	// Holder.Unit is not in scope in the controller.
	"/nest/unit": {"response": "nil"},
	// The file that declares the private Long sees it.
	"/nest/private-user": {"response.value": "object{own}"},
	// In the body of Holder, and in a subclass of it, the nested classes are in scope.
	"/holder/count": {"response": "object{own}"},
	"/holder/unit":  {"response": "string"},
	"/child/weight": {"response": "object{own}"},
}

func TestKotlinNestedAndPrivateClassesAreInScopeWhereKotlinPutsThem(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/nested-scope")
	checkTypeArgumentExport(t, endpoints, nestedScopeExpectation, nil)
	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// mixedScopeExpectation is the export of testdata/mixed-scope: a Java class of
// the package hides the standard Number for the Kotlin classes beside it, in a
// Kotlin controller and in a Java one alike.
var mixedScopeExpectation = map[string]map[string]string{
	"/java/dto": {
		"response.weight": "object{digits}", "response.count": "int", "response.weights": "array<object{digits}>",
	},
	"/java/save": {
		"request.weight": "object{digits}", "request.count": "int", "request.weights": "array<object{digits}>",
	},
	"/java/number": {"response": "object{digits}"},
	"/kotlin/dto": {
		"response.weight": "object{digits}", "response.count": "int", "response.weights": "array<object{digits}>",
	},
	"/kotlin/save": {
		"request.weight": "object{digits}", "request.count": "int", "request.weights": "array<object{digits}>",
	},
}

func TestAJavaClassOfThePackageHidesAKotlinNameForAJavaEndpointToo(t *testing.T) {
	abs, err := filepath.Abs("testdata/mixed-scope")
	if err != nil {
		t.Fatal(err)
	}
	c := New().(*KotlinCollector)
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: abs, NoDeps: true})
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	checkTypeArgumentExport(t, endpoints, mixedScopeExpectation, nil)
	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

// unitScopeExpectation is the export of testdata/unit-scope: the package
// declares a Unit, which is what a function that writes Unit returns, while a
// function written without a return type returns the standard Unit.
var unitScopeExpectation = map[string]map[string]string{
	"/unit/block":     {"response": "nil"},
	"/unit/qualified": {"response": "nil"},
	"/unit/enum":      {"response": "string"},
	"/unit/declared":  {"response": "string"},
}

func TestAFunctionWithoutAReturnTypeReturnsTheStandardUnit(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/unit-scope")
	checkTypeArgumentExport(t, endpoints, unitScopeExpectation, nil)
	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v, want none", unresolved)
	}
}

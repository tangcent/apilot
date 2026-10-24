package resolver

import (
	"reflect"
	"sort"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
	model "github.com/tangcent/apilot/api-model"
)

// The tests below pin the simple-name precedence ladder of JLS 6.5.5.1 and
// 6.4.1: a declaration in the same compilation unit shadows a single-type
// import, which shadows a declaration in the same package, which shadows an
// import-on-demand. Getting it wrong does not lose a field — it produces a
// plausible field set from the wrong class, so each case asserts the field
// names that must win and, where applicable, the ones that must not leak in.

// fieldNamesOf resolves typeName as written in source and returns the sorted
// field names of the resulting object.
func fieldNamesOf(t *testing.T, r *TypeResolver, source parser.Class, typeName string) []string {
	t.Helper()

	resolved := r.ResolveFrom(source, typeName, nil)
	if resolved == nil || resolved.Kind != model.KindObject {
		t.Fatalf("ResolveFrom(%q): expected an object model, got %+v", typeName, resolved)
	}

	names := make([]string, 0, len(resolved.Fields))
	for name := range resolved.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func assertFieldNames(t *testing.T, got, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("field set mismatch: got %v, want %v", got, want)
	}
}

func TestResolve_SameCompilationUnitShadowsImportOnDemand(t *testing.T) {
	const orderClientFile = "com/example/client/OrderClient.java"

	classes := []parser.Class{
		{
			Name:     "RefundVO",
			Package:  "com.example.apidto",
			FilePath: "com/example/apidto/RefundVO.java",
			Fields:   []parser.Field{{Name: "refundNo", Type: "String"}, {Name: "paymentId", Type: "Long"}},
		},
		{
			Name:     "OrderClient",
			Package:  "com.example.client",
			FilePath: orderClientFile,
			Imports:  []parser.Import{{Path: "com.example.apidto", OnDemand: true}},
		},
		{
			Name:     "RefundVO",
			Package:  "com.example.client",
			FilePath: orderClientFile,
			Fields:   []parser.Field{{Name: "orderId", Type: "Long"}},
		},
	}

	r := NewTypeResolver(classes)

	assertFieldNames(t, fieldNamesOf(t, r, classes[1], "RefundVO"), []string{"orderId"})
}

func TestResolve_SamePackageShadowsImportOnDemand(t *testing.T) {
	classes := []parser.Class{
		{
			Name:     "Wallet",
			Package:  "com.example.amodel",
			FilePath: "com/example/amodel/Wallet.java",
			Fields:   []parser.Field{{Name: "balance", Type: "BigDecimal"}},
		},
		{
			Name:     "WalletApi",
			Package:  "com.example.api",
			FilePath: "com/example/api/WalletApi.java",
			Imports:  []parser.Import{{Path: "com.example.amodel", OnDemand: true}},
		},
		{
			Name:     "Wallet",
			Package:  "com.example.api",
			FilePath: "com/example/api/Wallet.java",
			Fields:   []parser.Field{{Name: "address", Type: "String"}},
		},
	}

	r := NewTypeResolver(classes)

	assertFieldNames(t, fieldNamesOf(t, r, classes[1], "Wallet"), []string{"address"})
}

func TestResolve_SingleTypeImportShadowsSamePackage(t *testing.T) {
	classes := []parser.Class{
		{
			Name:     "Wallet",
			Package:  "com.example.model",
			FilePath: "com/example/model/Wallet.java",
			Fields:   []parser.Field{{Name: "balance", Type: "BigDecimal"}},
		},
		{
			Name:     "Wallet",
			Package:  "com.example.api",
			FilePath: "com/example/api/Wallet.java",
			Fields:   []parser.Field{{Name: "address", Type: "String"}},
		},
		{
			Name:     "WalletController",
			Package:  "com.example.api",
			FilePath: "com/example/api/WalletController.java",
			Imports:  []parser.Import{{Path: "com.example.model.Wallet"}},
		},
	}

	r := NewTypeResolver(classes)

	assertFieldNames(t, fieldNamesOf(t, r, classes[2], "Wallet"), []string{"balance"})
}

func TestResolve_ImportOnDemandResolvesWhenNothingCloserMatches(t *testing.T) {
	classes := []parser.Class{
		{
			Name:     "Order",
			Package:  "com.example.model",
			FilePath: "com/example/model/Order.java",
			Fields:   []parser.Field{{Name: "orderNo", Type: "String"}},
		},
		{
			Name:     "OrderQuery",
			Package:  "com.example.api",
			FilePath: "com/example/api/OrderQuery.java",
			Imports:  []parser.Import{{Path: "com.example.model", OnDemand: true}},
		},
	}

	r := NewTypeResolver(classes)

	assertFieldNames(t, fieldNamesOf(t, r, classes[1], "Order"), []string{"orderNo"})
}

// A single-type import names the type even when the imported body is not part
// of the scanned sources: a project class that merely shares the simple name
// must not be picked instead.
func TestResolve_SingleTypeImportShadowsSameNamedProjectClass(t *testing.T) {
	classes := []parser.Class{
		{
			Name:     "Date",
			Package:  "com.example.model",
			FilePath: "com/example/model/Date.java",
			Fields:   []parser.Field{{Name: "custom", Type: "String"}},
		},
		{
			Name:     "Event",
			Package:  "com.example.api",
			FilePath: "com/example/api/Event.java",
			Imports:  []parser.Import{{Path: "java.util.Date"}},
		},
	}

	r := NewTypeResolver(classes)

	resolved := r.ResolveFrom(classes[1], "Date", nil)
	if resolved.Kind != model.KindSingle || resolved.TypeName != model.JsonTypeString {
		t.Fatalf("expected java.util.Date to resolve to a string scalar, got %+v", resolved)
	}
}

// A qualified reference needs no scope: it names its own package.
func TestResolve_QualifiedNameSelectsItsOwnPackage(t *testing.T) {
	classes := []parser.Class{
		{
			Name:     "RefundVO",
			Package:  "com.example.apidto",
			FilePath: "com/example/apidto/RefundVO.java",
			Fields:   []parser.Field{{Name: "refundNo", Type: "String"}},
		},
		{
			Name:     "RefundVO",
			Package:  "com.example.client",
			FilePath: "com/example/client/OrderClient.java",
			Fields:   []parser.Field{{Name: "orderId", Type: "Long"}},
		},
	}

	r := NewTypeResolver(classes)

	assertFieldNames(t, fieldNamesOf(t, r, parser.Class{}, "com.example.apidto.RefundVO"), []string{"refundNo"})
	assertFieldNames(t, fieldNamesOf(t, r, parser.Class{}, "com.example.client.RefundVO"), []string{"orderId"})
}

// Unwrapping a wrapper is not a reason to forget the scope: the payload type is
// written in the same file as the wrapper, so it resolves the same way.
func TestResolve_WrapperKeepsCompilationUnitScope(t *testing.T) {
	const orderClientFile = "com/example/client/OrderClient.java"

	classes := []parser.Class{
		{
			Name:     "RefundVO",
			Package:  "com.example.apidto",
			FilePath: "com/example/apidto/RefundVO.java",
			Fields:   []parser.Field{{Name: "refundNo", Type: "String"}},
		},
		{
			Name:     "OrderClient",
			Package:  "com.example.client",
			FilePath: orderClientFile,
			Imports:  []parser.Import{{Path: "com.example.apidto", OnDemand: true}},
		},
		{
			Name:     "RefundVO",
			Package:  "com.example.client",
			FilePath: orderClientFile,
			Fields:   []parser.Field{{Name: "orderId", Type: "Long"}},
		},
	}

	r := NewTypeResolver(classes)

	for _, wrapper := range []string{"Mono<RefundVO>", "Flux<RefundVO>", "ResponseEntity<RefundVO>"} {
		t.Run(wrapper, func(t *testing.T) {
			resolved := r.ResolveFrom(classes[1], wrapper, nil)

			// Flux streams as an array; the other two carry a single payload.
			payload := resolved
			if resolved.Kind == model.KindArray {
				payload = resolved.Items
			}
			if payload == nil || payload.Kind != model.KindObject {
				t.Fatalf("expected an object payload, got %+v", resolved)
			}
			if _, isInline := payload.Fields["orderId"]; !isInline {
				t.Fatalf("expected the same-file RefundVO, got fields %v", payload.Fields)
			}
		})
	}
}

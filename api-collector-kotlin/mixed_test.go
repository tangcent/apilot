package kotlincollector

import (
	"testing"
)

func TestMixedTreeCollectsBothLanguages(t *testing.T) {
	endpoints, c := collectDir(t, "testdata/mixed")
	if len(endpoints) != 3 {
		t.Fatalf("expected 3 endpoints, got %d: %v", len(endpoints), endpointNames(endpoints))
	}

	// A Kotlin endpoint resolves a class written in Java.
	create := findEndpoint(t, endpoints, "create")
	address := objectField(t, create.RequestBody.Body, "address")
	objectField(t, address.Model, "city")
	objectField(t, address.Model, "street")
	objectField(t, findEndpoint(t, endpoints, "address").Response.Body, "street")

	// A Java endpoint resolves a class written in Kotlin.
	order := findEndpoint(t, endpoints, "order")
	if order.Folder != "LegacyController" || order.Path != "/legacy/order" {
		t.Fatalf("unexpected Java endpoint: %s %s", order.Folder, order.Path)
	}
	objectField(t, order.Response.Body, "orderNo")
	objectField(t, order.Response.Body, "amount")

	// An enum is a string of its constants whichever language the enum and the
	// endpoint that exports it are written in.
	checkEnumFields(t, "request body", create.RequestBody.Body.Fields, map[string]enumField{
		"status": {"string", []string{"ACTIVE: Open for orders.", "CLOSED: "}},
	})
	checkEnumFields(t, "response of order", order.Response.Body.Fields, map[string]enumField{
		"kind": {"string", []string{"RETAIL: Sold to consumers.", "WHOLESALE: "}},
	})

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Fatalf("unexpected unresolved types: %v", unresolved)
	}
	if frameworks := c.Frameworks(); len(frameworks) != 1 || frameworks[0] != "spring-mvc" {
		t.Fatalf("frameworks = %v", frameworks)
	}
}

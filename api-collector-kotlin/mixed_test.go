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

	if unresolved := c.Unresolved(); len(unresolved) != 0 {
		t.Fatalf("unexpected unresolved types: %v", unresolved)
	}
	if frameworks := c.Frameworks(); len(frameworks) != 1 || frameworks[0] != "spring-mvc" {
		t.Fatalf("frameworks = %v", frameworks)
	}
}

package javacollector

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

// TestCollect_SimpleNamePrecedence runs the whole pipeline over a project that
// deliberately declares the same simple name in more than one place. Picking
// the wrong declaration does not drop a field, it exports a plausible field
// set from the wrong class, so every case asserts both the fields that must
// appear and the ones that must not.
func TestCollect_SimpleNamePrecedence(t *testing.T) {
	sourceDir, err := filepath.Abs("testdata-shadowing")
	if err != nil {
		t.Fatalf("Failed to resolve testdata path: %v", err)
	}

	endpoints, err := New().Collect(collector.CollectContext{
		SourceDir: sourceDir,
		NoDeps:    true,
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	byName := make(map[string]*collector.ApiEndpoint, len(endpoints))
	for i := range endpoints {
		byName[endpoints[i].Name] = &endpoints[i]
	}

	tests := []struct {
		name       string
		endpoint   string
		response   bool
		wantFields []string
		wantAbsent []string
	}{
		{
			name:       "inline class shadows import-on-demand in a Feign client",
			endpoint:   "getRefundInfo",
			response:   true,
			wantFields: []string{"amount", "id", "orderId", "processedAt", "reason", "status"},
			wantAbsent: []string{"paymentId", "refundNo"},
		},
		{
			name:       "inline class shadows import-on-demand for a request body",
			endpoint:   "requestRefund",
			response:   false,
			wantFields: []string{"amount", "orderId", "reason"},
			wantAbsent: []string{"refundNo", "paymentId"},
		},
		{
			name:       "inline class shadows import-on-demand in a Spring controller",
			endpoint:   "getOrderStatistics",
			response:   true,
			wantFields: []string{"cancelledOrders", "completedOrders", "pendingOrders", "totalAmount", "totalOrders"},
			wantAbsent: []string{"averageOrderValue"},
		},
		{
			name:       "single-type import shadows the same package",
			endpoint:   "getWallet",
			response:   true,
			wantFields: []string{"balance", "id"},
			wantAbsent: []string{"address"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ep, found := byName[tt.endpoint]
			if !found {
				t.Fatalf("endpoint %q was not collected", tt.endpoint)
			}

			var body *model.ObjectModel
			if tt.response {
				if ep.Response == nil {
					t.Fatalf("endpoint %q has no response schema", tt.endpoint)
				}
				body = ep.Response.Body
			} else {
				if ep.RequestBody == nil {
					t.Fatalf("endpoint %q has no request body schema", tt.endpoint)
				}
				body = ep.RequestBody.Body
			}

			got := sortedFieldNames(t, tt.endpoint, body)
			if !reflect.DeepEqual(got, tt.wantFields) {
				t.Fatalf("field set mismatch: got %v, want %v", got, tt.wantFields)
			}

			for _, absent := range tt.wantAbsent {
				if body.Fields[absent] != nil {
					t.Errorf("field %q belongs to the shadowed declaration but was exported", absent)
				}
			}
		})
	}
}

func sortedFieldNames(t *testing.T, endpoint string, body *model.ObjectModel) []string {
	t.Helper()

	if body == nil {
		t.Fatalf("endpoint %q exported no schema", endpoint)
	}
	if body.Kind != model.KindObject {
		t.Fatalf("endpoint %q exported kind %q instead of an object", endpoint, body.Kind)
	}

	names := make([]string, 0, len(body.Fields))
	for name := range body.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

package kotlincollector

import (
	"path/filepath"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func TestRequiredFieldsModes(t *testing.T) {
	abs, err := filepath.Abs("testdata/required")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		mode     string
		request  map[string]bool
		response map[string]bool
	}{
		{
			mode:     collector.RequiredFieldsDefault,
			request:  map[string]bool{"orderNo": true, "amount": true, "remark": true, "coupon": false},
			response: map[string]bool{"orderNo": true, "status": true},
		},
		{
			mode:     collector.RequiredFieldsConstraints,
			request:  map[string]bool{"orderNo": true, "amount": true, "remark": false, "coupon": false},
			response: map[string]bool{"orderNo": false, "status": true},
		},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			endpoints, err := New().Collect(collector.CollectContext{
				SourceDir: abs,
				NoDeps:    true,
				Config:    map[string]string{collector.ConfigRequiredFields: tc.mode},
			})
			if err != nil {
				t.Fatal(err)
			}
			ep := findEndpoint(t, endpoints, "create")
			for name, want := range tc.request {
				if got := objectField(t, ep.RequestBody.Body, name).Required; got != want {
					t.Errorf("request %s: required = %v, want %v", name, got, want)
				}
			}
			for name, want := range tc.response {
				if got := objectField(t, ep.Response.Body, name).Required; got != want {
					t.Errorf("response %s: required = %v, want %v", name, got, want)
				}
			}
		})
	}
}

func TestRequiredFieldsRejectsUnknownMode(t *testing.T) {
	abs, err := filepath.Abs("testdata/required")
	if err != nil {
		t.Fatal(err)
	}
	_, err = New().Collect(collector.CollectContext{
		SourceDir: abs,
		Config:    map[string]string{collector.ConfigRequiredFields: "strict"},
	})
	if err == nil {
		t.Fatal("an unknown mode should fail the collection")
	}
}

package resolver

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestResolve_RequiredFieldsModes(t *testing.T) {
	classes := []parser.Class{{
		Name: "CreateOrderRequest",
		Fields: []parser.Field{
			{Name: "orderNo", Type: "String", Annotations: []parser.Annotation{{Name: "NotBlank"}}},
			{Name: "amount", Type: "Long", Annotations: []parser.Annotation{{Name: "NotNull"}}},
			{Name: "items", Type: "List<String>", Annotations: []parser.Annotation{{Name: "NotEmpty"}}},
			{Name: "remark", Type: "String"},
			{Name: "coupon", Type: "String", Annotations: []parser.Annotation{{Name: "Nullable"}}},
			{Name: "note", Type: "String", Annotations: []parser.Annotation{{Name: "Size", Params: map[string]string{"max": "64"}}}},
		},
	}}

	for _, tc := range []struct {
		mode string
		want map[string]bool
	}{
		{
			// Required unless nullable.
			mode: collector.RequiredFieldsDefault,
			want: map[string]bool{"orderNo": true, "amount": true, "items": true, "remark": true, "coupon": false, "note": true},
		},
		{
			// An empty mode is the default.
			mode: "",
			want: map[string]bool{"orderNo": true, "amount": true, "items": true, "remark": true, "coupon": false, "note": true},
		},
		{
			// Required only when a constraint demands a value; @Size does not.
			mode: collector.RequiredFieldsConstraints,
			want: map[string]bool{"orderNo": true, "amount": true, "items": true, "remark": false, "coupon": false, "note": false},
		},
	} {
		t.Run("mode="+tc.mode, func(t *testing.T) {
			r := NewTypeResolver(classes)
			r.SetRequiredFields(tc.mode)
			result := r.Resolve("CreateOrderRequest", nil)
			for name, want := range tc.want {
				field := result.Fields[name]
				if field == nil {
					t.Fatalf("field %q missing", name)
				}
				if field.Required != want {
					t.Errorf("%s: required = %v, want %v", name, field.Required, want)
				}
			}
		})
	}
}

func TestResolve_RequiredFieldsDocumentationWins(t *testing.T) {
	// An explicit required flag in a documentation annotation overrides the mode.
	classes := []parser.Class{{
		Name: "Query",
		Fields: []parser.Field{
			{Name: "keyword", Type: "String", Annotations: []parser.Annotation{
				{Name: "Schema", Params: map[string]string{"required": "true"}},
			}},
		},
	}}
	r := NewTypeResolver(classes)
	r.SetRequiredFields(collector.RequiredFieldsConstraints)
	if !r.Resolve("Query", nil).Fields["keyword"].Required {
		t.Fatal("@Schema(required = true) should make the field required in constraints mode")
	}
}

package collector

import (
	"strings"
	"testing"
)

func TestCollectContextRequiredFields(t *testing.T) {
	for _, tc := range []struct {
		config map[string]string
		want   string
	}{
		{nil, RequiredFieldsDefault},
		{map[string]string{ConfigRequiredFields: ""}, RequiredFieldsDefault},
		{map[string]string{ConfigRequiredFields: "default"}, RequiredFieldsDefault},
		{map[string]string{ConfigRequiredFields: "constraints"}, RequiredFieldsConstraints},
	} {
		got, err := CollectContext{Config: tc.config}.RequiredFields()
		if err != nil || got != tc.want {
			t.Errorf("RequiredFields(%v) = %q, %v; want %q", tc.config, got, err, tc.want)
		}
	}

	_, err := CollectContext{Config: map[string]string{ConfigRequiredFields: "strict"}}.RequiredFields()
	if err == nil || !strings.Contains(err.Error(), `"strict"`) || !strings.Contains(err.Error(), "constraints") {
		t.Fatalf("an unknown mode should fail and name the accepted ones, got %v", err)
	}
}

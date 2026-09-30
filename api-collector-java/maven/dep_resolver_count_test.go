package maven

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestMavenDependencyResolver_ResolvedCount(t *testing.T) {
	r := &MavenDependencyResolver{
		cache: map[string]*parser.Class{
			"CommonUser": {Name: "CommonUser"},
			"PageResult": {Name: "PageResult"},
		},
		misses: map[string]bool{"Unknown": true},
	}

	if got := r.ResolvedCount(); got != 2 {
		t.Errorf("expected ResolvedCount 2, got %d", got)
	}
}

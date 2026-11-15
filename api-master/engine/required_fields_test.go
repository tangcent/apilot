package engine

import (
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

// contextCapturingCollector records the context it was asked to collect with.
type contextCapturingCollector struct {
	name string
	got  *collector.CollectContext
}

func (c *contextCapturingCollector) Name() string                 { return c.name }
func (c *contextCapturingCollector) SupportedLanguages() []string { return nil }
func (c *contextCapturingCollector) Collect(ctx collector.CollectContext) ([]collector.ApiEndpoint, error) {
	c.got = &ctx
	return []collector.ApiEndpoint{{Name: "ping", Method: "GET", Path: "/ping"}}, nil
}

func TestRun_RequiredFieldsReachesCollector(t *testing.T) {
	c := &contextCapturingCollector{name: "required-fields-capture"}
	RegisterCollector(c)
	RegisterFormatter(&mockFormatter{name: "required-fields-out", formatOutput: []byte("ok")})

	err := Run(Config{
		SourceDir:      t.TempDir(),
		CollectorName:  c.name,
		FormatterName:  "required-fields-out",
		OutputPath:     t.TempDir() + "/out.json",
		RequiredFields: collector.RequiredFieldsConstraints,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if c.got == nil {
		t.Fatal("collector was not called")
	}
	if mode, _ := c.got.RequiredFields(); mode != collector.RequiredFieldsConstraints {
		t.Fatalf("collector saw mode %q, want %q", mode, collector.RequiredFieldsConstraints)
	}
}

func TestRun_RequiredFieldsRejectsUnknownMode(t *testing.T) {
	c := &contextCapturingCollector{name: "required-fields-reject"}
	RegisterCollector(c)

	err := Run(Config{
		SourceDir:      t.TempDir(),
		CollectorName:  c.name,
		RequiredFields: "strict",
	})
	if err == nil || !strings.Contains(err.Error(), "unknown required-fields mode") {
		t.Fatalf("expected an unknown-mode error, got %v", err)
	}
	if c.got != nil {
		t.Fatal("the tree should not be collected with an unknown mode")
	}
}

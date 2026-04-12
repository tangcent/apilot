package javacollector

import (
	"path/filepath"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

func TestCollect_SpringMVC(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints from Spring MVC controller")
	}

	found := false
	for _, ep := range endpoints {
		if ep.Folder == "UserController" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected endpoints from UserController")
	}
}

func TestCollect_JAXRS(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"jaxrs"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints from JAX-RS resource")
	}

	found := false
	for _, ep := range endpoints {
		if ep.Folder == "UserResource" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected endpoints from UserResource")
	}
}

func TestCollect_Feign(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"feign"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints from Feign client")
	}

	found := false
	for _, ep := range endpoints {
		if ep.Folder == "UserClient" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected endpoints from UserClient")
	}
}

func TestCollect_AllFrameworks(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	// No framework hints = detect all
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: testdataDir,
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Should find endpoints from Spring MVC (5) + JAX-RS (5) + Feign (4) = 14
	if len(endpoints) < 14 {
		t.Errorf("Expected at least 14 endpoints, got %d", len(endpoints))
	}

	protocols := make(map[string]bool)
	for _, ep := range endpoints {
		protocols[ep.Protocol] = true
	}
	if !protocols["http"] {
		t.Error("Expected http protocol endpoints")
	}
}

func TestCollect_EndpointFields(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	for _, ep := range endpoints {
		if ep.Protocol != "http" {
			t.Errorf("Expected protocol 'http', got '%s'", ep.Protocol)
		}
		if ep.Method == "" {
			t.Errorf("Expected non-empty method for endpoint %s", ep.Name)
		}
		if ep.Path == "" {
			t.Errorf("Expected non-empty path for endpoint %s", ep.Name)
		}
	}
}

func TestCollect_FrameworkAliases(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	for _, alias := range []string{"spring", "spring-mvc", "springmvc"} {
		endpoints, err := c.Collect(collector.CollectContext{
			SourceDir:  testdataDir,
			Frameworks: []string{alias},
		})
		if err != nil {
			t.Fatalf("Collect with alias '%s' failed: %v", alias, err)
		}
		if len(endpoints) == 0 {
			t.Errorf("Expected endpoints for alias '%s'", alias)
		}
	}
}

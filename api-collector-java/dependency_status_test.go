package javacollector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
)

const pomWithDeps = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
	<modelVersion>4.0.0</modelVersion>
	<groupId>com.example</groupId>
	<artifactId>demo</artifactId>
	<version>1.0.0</version>
	<dependencies>
		<dependency>
			<groupId>org.apache.commons</groupId>
			<artifactId>commons-lang3</artifactId>
			<version>3.14.0</version>
		</dependency>
		<dependency>
			<groupId>com.google.guava</groupId>
			<artifactId>guava</artifactId>
			<version>33.0.0-jre</version>
		</dependency>
	</dependencies>
</project>
`

const pomWithoutDeps = `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
	<modelVersion>4.0.0</modelVersion>
	<groupId>com.example</groupId>
	<artifactId>demo</artifactId>
	<version>1.0.0</version>
</project>
`

func writePom(t *testing.T, dir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "pom.xml"), []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write pom.xml: %v", err)
	}
}

func TestDependencyResolution_Disabled(t *testing.T) {
	c := New().(*JavaCollector)
	_, err := c.Collect(collector.CollectContext{SourceDir: "testdata", NoDeps: true})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionDisabled {
		t.Errorf("expected state disabled, got %q", dr.State)
	}
}

func TestDependencyResolution_NoBuildFileIsNoDeps(t *testing.T) {
	c := New().(*JavaCollector)

	// testdata holds only Java sources, no pom.xml or build.gradle.
	_, err := c.Collect(collector.CollectContext{SourceDir: "testdata"})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionNoDeps {
		t.Errorf("expected state no-deps, got %q", dr.State)
	}
}

func TestDependencyResolution_EmptyPomIsNoDeps(t *testing.T) {
	dir := t.TempDir()
	writePom(t, dir, pomWithoutDeps)

	c := New().(*JavaCollector)
	_, err := c.Collect(collector.CollectContext{SourceDir: dir})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionNoDeps {
		t.Errorf("expected state no-deps, got %q", dr.State)
	}
}

func TestDependencyResolution_ToolMissingWarnsWithGuidance(t *testing.T) {
	dir := t.TempDir()
	writePom(t, dir, pomWithDeps)

	orig := mavenAvailable
	mavenAvailable = func() bool { return false }
	defer func() { mavenAvailable = orig }()

	c := New().(*JavaCollector)
	_, err := c.Collect(collector.CollectContext{SourceDir: dir})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionToolMissing {
		t.Fatalf("expected state tool-missing, got %q", dr.State)
	}
	if !strings.Contains(dr.Detail, "maven-indexer-cli") {
		t.Errorf("detail should name the missing tool, got %q", dr.Detail)
	}
	if !strings.Contains(dr.Detail, "2 declared dependencies") {
		t.Errorf("detail should report the declared dependency count, got %q", dr.Detail)
	}
	if !strings.Contains(dr.Detail, "https://github.com/tangcent/maven-indexer-cli") {
		t.Errorf("detail should carry install guidance, got %q", dr.Detail)
	}
}

func TestDependencyResolution_ToolPresentIsActive(t *testing.T) {
	dir := t.TempDir()
	writePom(t, dir, pomWithDeps)

	orig := mavenAvailable
	mavenAvailable = func() bool { return true }
	defer func() { mavenAvailable = orig }()

	c := New().(*JavaCollector)
	// The fixture has no Java sources, so Collect returns no endpoints; the
	// resolver is still constructed and the run reported as active with zero
	// types resolved (none were requested).
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: dir})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if len(endpoints) != 0 {
		t.Fatalf("expected no endpoints from an empty project, got %d", len(endpoints))
	}

	dr := c.DependencyResolution()
	if dr.State != collector.DependencyResolutionActive {
		t.Errorf("expected state active, got %q", dr.State)
	}
	if dr.ResolvedTypes != 0 {
		t.Errorf("expected 0 resolved types, got %d", dr.ResolvedTypes)
	}
}

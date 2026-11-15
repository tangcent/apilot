package javacollector

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/maven"
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
	if !strings.Contains(dr.Detail, "npm install -g maven-indexer-cli") {
		t.Errorf("detail should carry install guidance, got %q", dr.Detail)
	}
}

// stubIndexer makes the CLI look installed and answer project-context with
// project, or with err.
func stubIndexer(t *testing.T, project maven.ProjectContext, err error) {
	t.Helper()
	origAvailable, origContext := mavenAvailable, loadProjectContext
	mavenAvailable = func() bool { return true }
	loadProjectContext = func(string) (maven.ProjectContext, error) { return project, err }
	t.Cleanup(func() { mavenAvailable, loadProjectContext = origAvailable, origContext })
}

func collectStatus(t *testing.T, dir string) collector.DependencyResolution {
	t.Helper()
	c := New().(*JavaCollector)
	if _, err := c.Collect(collector.CollectContext{SourceDir: dir}); err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	return c.DependencyResolution()
}

const pomAggregator = `<project><modelVersion>4.0.0</modelVersion>
	<groupId>com.example</groupId><artifactId>parent</artifactId><version>1.0.0</version>
	<packaging>pom</packaging>
	<modules><module>api</module><module>server</module></modules>
</project>`

func TestDependencyResolution_AnAggregatorRootIsAskedBeforeItIsJudgedToHaveNoDeps(t *testing.T) {
	dir := t.TempDir()
	writePom(t, dir, pomAggregator)

	stubIndexer(t, maven.ProjectContext{Kind: "maven", Tree: "resolved", Resolved: 344}, nil)
	if dr := collectStatus(t, dir); dr.State != collector.DependencyResolutionActive || dr.Detail != "" {
		t.Errorf("state = %q, detail %q; want active: the modules' tree has dependencies", dr.State, dr.Detail)
	}

	stubIndexer(t, maven.ProjectContext{Kind: "maven", Tree: "resolved"}, nil)
	if dr := collectStatus(t, dir); dr.State != collector.DependencyResolutionNoDeps {
		t.Errorf("state = %q, want no-deps: nothing in the tree", dr.State)
	}
}

func TestDependencyResolution_AnAggregatorRootWithoutTheToolNamesItsModules(t *testing.T) {
	dir := t.TempDir()
	writePom(t, dir, pomAggregator)

	orig := mavenAvailable
	mavenAvailable = func() bool { return false }
	defer func() { mavenAvailable = orig }()

	dr := collectStatus(t, dir)
	if dr.State != collector.DependencyResolutionToolMissing || !strings.Contains(dr.Detail, "the dependencies of its modules") {
		t.Errorf("state = %q, detail %q", dr.State, dr.Detail)
	}
}

func TestDependencyResolution_ACLIWithoutProjectContextAsksForAnUpgrade(t *testing.T) {
	dir := t.TempDir()
	writePom(t, dir, pomWithDeps)
	stubIndexer(t, maven.ProjectContext{}, errors.New("unknown command 'project-context'"))

	dr := collectStatus(t, dir)
	if dr.State != collector.DependencyResolutionToolMissing || !strings.Contains(dr.Detail, "1.2.0 or later") {
		t.Errorf("state = %q, detail %q", dr.State, dr.Detail)
	}
}

func TestDependencyResolution_AnUnresolvedTreeIsExplained(t *testing.T) {
	dir := t.TempDir()
	writePom(t, dir, pomWithDeps)
	stubIndexer(t, maven.ProjectContext{Kind: "maven", Tree: "declared", Declared: 2, ResolveError: "mvn exited with code 1"}, nil)

	dr := collectStatus(t, dir)
	if dr.State != collector.DependencyResolutionActive {
		t.Fatalf("state = %q, want active: the declared versions still pin", dr.State)
	}
	if !strings.Contains(dr.Detail, "mvn exited with code 1") || !strings.Contains(dr.Detail, "only the 2 dependencies") {
		t.Errorf("detail = %q", dr.Detail)
	}
}

func TestDependencyResolution_SharedDependenciesAreNotSetUpAgain(t *testing.T) {
	dir := t.TempDir()
	writePom(t, dir, pomWithDeps)
	// Setting up a resolution of its own would ask the indexer.
	stubIndexer(t, maven.ProjectContext{}, errors.New("the indexer must not be asked"))

	c := New().(*JavaCollector)
	c.ShareDependencies(&Dependencies{resolution: collector.DependencyResolution{State: collector.DependencyResolutionActive}})
	if dr := collectStatus(t, dir); dr.State != collector.DependencyResolutionToolMissing {
		t.Fatalf("a collector without shared dependencies sets up its own: state = %q", dr.State)
	}
	if _, err := c.Collect(collector.CollectContext{SourceDir: dir}); err != nil {
		t.Fatal(err)
	}
	if dr := c.DependencyResolution(); dr.State != collector.DependencyResolutionActive {
		t.Errorf("state = %q, want the shared resolution", dr.State)
	}
}

func TestDependencyResolution_ToolPresentIsActive(t *testing.T) {
	dir := t.TempDir()
	writePom(t, dir, pomWithDeps)

	stubIndexer(t, maven.ProjectContext{Kind: "maven", Tree: "resolved", Declared: 2, Resolved: 40}, nil)

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

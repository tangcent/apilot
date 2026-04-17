package maven

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParsePomXML(t *testing.T) {
	deps, err := parsePomXML(filepath.Join("testdata", "pom.xml"))
	if err != nil {
		t.Fatalf("parsePomXML failed: %v", err)
	}

	if len(deps) != 2 {
		t.Fatalf("Expected 2 dependencies, got %d", len(deps))
	}

	if deps[0].GroupID != "org.springframework.boot" {
		t.Errorf("Expected groupId 'org.springframework.boot', got '%s'", deps[0].GroupID)
	}
	if deps[0].ArtifactID != "spring-boot-starter-web" {
		t.Errorf("Expected artifactId 'spring-boot-starter-web', got '%s'", deps[0].ArtifactID)
	}
	if deps[0].Version != "3.2.0" {
		t.Errorf("Expected version '3.2.0', got '%s'", deps[0].Version)
	}

	if deps[1].GroupID != "com.google.guava" {
		t.Errorf("Expected groupId 'com.google.guava', got '%s'", deps[1].GroupID)
	}
	if deps[1].ArtifactID != "guava" {
		t.Errorf("Expected artifactId 'guava', got '%s'", deps[1].ArtifactID)
	}
}

func TestParsePomXML_EmptyDeps(t *testing.T) {
	dir := t.TempDir()
	pomPath := filepath.Join(dir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(`<project><dependencies></dependencies></project>`), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := parsePomXML(pomPath)
	if err != nil {
		t.Fatalf("parsePomXML failed: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(deps))
	}
}

func TestParsePomXML_NoDepsSection(t *testing.T) {
	dir := t.TempDir()
	pomPath := filepath.Join(dir, "pom.xml")
	if err := os.WriteFile(pomPath, []byte(`<project></project>`), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := parsePomXML(pomPath)
	if err != nil {
		t.Fatalf("parsePomXML failed: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(deps))
	}
}

func TestParsePomXML_SkipsInvalidDeps(t *testing.T) {
	dir := t.TempDir()
	pomPath := filepath.Join(dir, "pom.xml")
	content := `<project><dependencies>
		<dependency><groupId>com.example</groupId><artifactId>valid</artifactId><version>1.0</version></dependency>
		<dependency><groupId></groupId><artifactId>missing-group</artifactId></dependency>
		<dependency><groupId>com.example</groupId><artifactId></artifactId></dependency>
	</dependencies></project>`
	if err := os.WriteFile(pomPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := parsePomXML(pomPath)
	if err != nil {
		t.Fatalf("parsePomXML failed: %v", err)
	}
	if len(deps) != 1 {
		t.Errorf("Expected 1 valid dependency, got %d", len(deps))
	}
	if deps[0].ArtifactID != "valid" {
		t.Errorf("Expected artifactId 'valid', got '%s'", deps[0].ArtifactID)
	}
}

func TestParseGradleFile(t *testing.T) {
	deps, err := parseGradleFile(filepath.Join("testdata", "build.gradle"))
	if err != nil {
		t.Fatalf("parseGradleFile failed: %v", err)
	}

	if len(deps) != 5 {
		t.Fatalf("Expected 5 dependencies, got %d", len(deps))
	}

	if deps[0].GroupID != "org.springframework.boot" {
		t.Errorf("Expected groupId 'org.springframework.boot', got '%s'", deps[0].GroupID)
	}
	if deps[0].ArtifactID != "spring-boot-starter-web" {
		t.Errorf("Expected artifactId 'spring-boot-starter-web', got '%s'", deps[0].ArtifactID)
	}
	if deps[0].Version != "3.2.0" {
		t.Errorf("Expected version '3.2.0', got '%s'", deps[0].Version)
	}

	if deps[2].GroupID != "org.projectlombok" {
		t.Errorf("Expected groupId 'org.projectlombok', got '%s'", deps[2].GroupID)
	}
	if deps[2].ArtifactID != "lombok" {
		t.Errorf("Expected artifactId 'lombok', got '%s'", deps[2].ArtifactID)
	}
}

func TestParseGradleFile_Kts(t *testing.T) {
	deps, err := parseGradleFile(filepath.Join("testdata", "build.gradle.kts"))
	if err != nil {
		t.Fatalf("parseGradleFile failed: %v", err)
	}

	if len(deps) != 2 {
		t.Fatalf("Expected 2 dependencies, got %d", len(deps))
	}

	if deps[0].GroupID != "org.springframework.boot" {
		t.Errorf("Expected groupId 'org.springframework.boot', got '%s'", deps[0].GroupID)
	}
}

func TestParseGradleFile_Empty(t *testing.T) {
	dir := t.TempDir()
	gradlePath := filepath.Join(dir, "build.gradle")
	if err := os.WriteFile(gradlePath, []byte(`dependencies { }`), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := parseGradleFile(gradlePath)
	if err != nil {
		t.Fatalf("parseGradleFile failed: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies, got %d", len(deps))
	}
}

func TestDetectDependencies_PomXML(t *testing.T) {
	deps, err := detectDependencies("testdata")
	if err != nil {
		t.Fatalf("detectDependencies failed: %v", err)
	}
	if len(deps) == 0 {
		t.Error("Expected dependencies from pom.xml")
	}
}

func TestDetectDependencies_Gradle(t *testing.T) {
	dir := t.TempDir()
	gradlePath := filepath.Join(dir, "build.gradle")
	content := `dependencies { implementation 'com.example:mylib:1.0.0' }`
	if err := os.WriteFile(gradlePath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	deps, err := detectDependencies(dir)
	if err != nil {
		t.Fatalf("detectDependencies failed: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}
	if deps[0].ArtifactID != "mylib" {
		t.Errorf("Expected artifactId 'mylib', got '%s'", deps[0].ArtifactID)
	}
}

func TestDetectDependencies_NoBuildFile(t *testing.T) {
	dir := t.TempDir()

	deps, err := detectDependencies(dir)
	if err != nil {
		t.Fatalf("detectDependencies failed: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies with no build file, got %d", len(deps))
	}
}

func TestHasBuildFile(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(dir string)
		expected bool
	}{
		{
			name:     "no build file",
			setup:    func(dir string) {},
			expected: false,
		},
		{
			name:     "pom.xml exists",
			setup:    func(dir string) { os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0644) },
			expected: true,
		},
		{
			name:     "build.gradle exists",
			setup:    func(dir string) { os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(""), 0644) },
			expected: true,
		},
		{
			name:     "build.gradle.kts exists",
			setup:    func(dir string) { os.WriteFile(filepath.Join(dir, "build.gradle.kts"), []byte(""), 0644) },
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			tt.setup(dir)
			result := HasBuildFile(dir)
			if result != tt.expected {
				t.Errorf("HasBuildFile() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestHasBuildFile_Testdata(t *testing.T) {
	if !HasBuildFile("testdata") {
		t.Error("Expected HasBuildFile to return true for testdata directory")
	}
}

func TestFindBestMatch(t *testing.T) {
	artifacts := []Artifact{
		{GroupID: "org.other", ArtifactID: "spring-web", Version: "6.1.0"},
		{GroupID: "org.springframework", ArtifactID: "spring-web", Version: "6.1.0"},
		{GroupID: "org.springframework", ArtifactID: "spring-web", Version: "6.0.0"},
	}

	tests := []struct {
		name     string
		dep      Dependency
		expected string
	}{
		{
			name:     "exact version match",
			dep:      Dependency{GroupID: "org.springframework", ArtifactID: "spring-web", Version: "6.0.0"},
			expected: "6.0.0",
		},
		{
			name:     "group match without version",
			dep:      Dependency{GroupID: "org.springframework", ArtifactID: "spring-web"},
			expected: "6.1.0",
		},
		{
			name:     "no group match falls back to first",
			dep:      Dependency{GroupID: "com.unknown", ArtifactID: "spring-web"},
			expected: "6.1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := findBestMatch(artifacts, tt.dep)
			if matched == nil {
				t.Fatal("Expected non-nil match")
			}
			if matched.Version != tt.expected {
				t.Errorf("Expected version '%s', got '%s'", tt.expected, matched.Version)
			}
		})
	}
}

func TestFindBestMatch_EmptyArtifacts(t *testing.T) {
	dep := Dependency{GroupID: "com.example", ArtifactID: "mylib"}
	matched := findBestMatch(nil, dep)
	if matched != nil {
		t.Error("Expected nil match for empty artifacts")
	}
}

func TestResolveJarPath(t *testing.T) {
	dir := t.TempDir()
	jarName := "mylib-1.0.0.jar"
	jarPath := filepath.Join(dir, jarName)
	if err := os.WriteFile(jarPath, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	artifact := &Artifact{
		ArtifactID: "mylib",
		Version:    "1.0.0",
		AbsPath:    dir,
	}

	result := resolveJarPath(artifact)
	if result != jarPath {
		t.Errorf("Expected '%s', got '%s'", jarPath, result)
	}
}

func TestResolveJarPath_JarNotFound(t *testing.T) {
	artifact := &Artifact{
		ArtifactID: "mylib",
		Version:    "1.0.0",
		AbsPath:    "/nonexistent/path",
	}

	result := resolveJarPath(artifact)
	if result != "/nonexistent/path" {
		t.Errorf("Expected abspath fallback '/nonexistent/path', got '%s'", result)
	}
}

func TestResolveJarPath_EmptyAbsPath(t *testing.T) {
	artifact := &Artifact{
		ArtifactID: "mylib",
		Version:    "1.0.0",
		AbsPath:    "",
	}

	result := resolveJarPath(artifact)
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

func TestResolve_CLIUnavailable(t *testing.T) {
	dir := t.TempDir()

	originalPath := os.Getenv("PATH")
	os.Setenv("PATH", t.TempDir())
	defer os.Setenv("PATH", originalPath)

	jarPaths, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve should not error when CLI unavailable: %v", err)
	}
	if jarPaths != nil {
		t.Errorf("Expected nil jarPaths when CLI unavailable, got %v", jarPaths)
	}
}

func TestResolve_NoBuildFile(t *testing.T) {
	dir := t.TempDir()

	jarPaths, err := Resolve(dir)
	if err != nil {
		t.Fatalf("Resolve should not error without build file: %v", err)
	}
	if jarPaths != nil {
		t.Errorf("Expected nil jarPaths without build file, got %v", jarPaths)
	}
}

func TestArtifactJSONParsing(t *testing.T) {
	jsonData := `[{
		"groupId": "org.springframework",
		"artifactId": "spring-web",
		"version": "6.1.0",
		"abspath": "/Users/test/.m2/repository/org/springframework/spring-web/6.1.0",
		"hasSource": true
	}]`

	var artifacts []Artifact
	if err := json.Unmarshal([]byte(jsonData), &artifacts); err != nil {
		t.Fatalf("Failed to parse artifact JSON: %v", err)
	}

	if len(artifacts) != 1 {
		t.Fatalf("Expected 1 artifact, got %d", len(artifacts))
	}

	a := artifacts[0]
	if a.GroupID != "org.springframework" {
		t.Errorf("Expected groupId 'org.springframework', got '%s'", a.GroupID)
	}
	if a.ArtifactID != "spring-web" {
		t.Errorf("Expected artifactId 'spring-web', got '%s'", a.ArtifactID)
	}
	if a.Version != "6.1.0" {
		t.Errorf("Expected version '6.1.0', got '%s'", a.Version)
	}
	if a.AbsPath != "/Users/test/.m2/repository/org/springframework/spring-web/6.1.0" {
		t.Errorf("Unexpected abspath: '%s'", a.AbsPath)
	}
	if !a.HasSource {
		t.Error("Expected hasSource=true")
	}
}

func TestResolveDependencies_Deduplication(t *testing.T) {
	deps := []Dependency{
		{GroupID: "com.example", ArtifactID: "mylib", Version: "1.0.0"},
		{GroupID: "com.example", ArtifactID: "mylib", Version: "1.0.0"},
	}

	seen := make(map[string]bool)
	var jarPaths []string

	for _, dep := range deps {
		jarPath := dep.GroupID + ":" + dep.ArtifactID + ":" + dep.Version
		if !seen[jarPath] {
			seen[jarPath] = true
			jarPaths = append(jarPaths, jarPath)
		}
	}

	if len(jarPaths) != 1 {
		t.Errorf("Expected 1 unique JAR path after dedup, got %d", len(jarPaths))
	}
}

// Package maven integrates with maven-indexer-cli to resolve dependency JARs.
// See: https://github.com/tangcent/maven-indexer-cli
package maven

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const cliName = "maven-indexer-cli"

// Dependency represents a Maven/Gradle dependency coordinate.
type Dependency struct {
	GroupID    string
	ArtifactID string
	Version    string
}

// Artifact represents a resolved artifact from maven-indexer-cli.
type Artifact struct {
	GroupID    string `json:"groupId"`
	ArtifactID string `json:"artifactId"`
	Version    string `json:"version"`
	AbsPath    string `json:"abspath"`
	HasSource  bool   `json:"hasSource"`
}

// Resolve attempts to resolve Maven/Gradle dependency coordinates in sourceDir
// by invoking maven-indexer-cli as a subprocess.
// Returns the list of resolved JAR paths for type analysis.
// If maven-indexer-cli is not available, returns nil without error.
func Resolve(sourceDir string) ([]string, error) {
	if !isCLIAvailable() {
		return nil, nil
	}

	deps, err := detectDependencies(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to detect dependencies: %w", err)
	}

	if len(deps) == 0 {
		return nil, nil
	}

	return resolveDependencies(deps)
}

// isCLIAvailable checks whether maven-indexer-cli is on PATH.
func isCLIAvailable() bool {
	_, err := exec.LookPath(cliName)
	return err == nil
}

// detectDependencies detects dependencies from pom.xml or build.gradle in sourceDir.
func detectDependencies(sourceDir string) ([]Dependency, error) {
	pomPath := filepath.Join(sourceDir, "pom.xml")
	if _, err := os.Stat(pomPath); err == nil {
		return parsePomXML(pomPath)
	}

	for _, gradleFile := range []string{"build.gradle", "build.gradle.kts"} {
		gradlePath := filepath.Join(sourceDir, gradleFile)
		if _, err := os.Stat(gradlePath); err == nil {
			return parseGradleFile(gradlePath)
		}
	}

	return nil, nil
}

// pomXML represents the relevant parts of a Maven pom.xml file.
type pomXML struct {
	Dependencies struct {
		Dependency []pomDependency `xml:"dependency"`
	} `xml:"dependencies"`
}

// pomDependency represents a single dependency entry in pom.xml.
type pomDependency struct {
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

// parsePomXML extracts dependencies from a Maven pom.xml file.
func parsePomXML(path string) ([]Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read pom.xml: %w", err)
	}

	var pom pomXML
	if err := xml.Unmarshal(data, &pom); err != nil {
		return nil, fmt.Errorf("failed to parse pom.xml: %w", err)
	}

	deps := make([]Dependency, 0, len(pom.Dependencies.Dependency))
	for _, d := range pom.Dependencies.Dependency {
		if d.GroupID == "" || d.ArtifactID == "" {
			continue
		}
		deps = append(deps, Dependency{
			GroupID:    d.GroupID,
			ArtifactID: d.ArtifactID,
			Version:    d.Version,
		})
	}

	return deps, nil
}

var gradleDepRe = regexp.MustCompile(
	`(?:implementation|api|compileOnly|runtimeOnly|testImplementation)\s*[( ]["']([^"':]+):([^"':]+):([^"':]+)["']`,
)

// parseGradleFile extracts dependencies from a Gradle build file.
func parseGradleFile(path string) ([]Dependency, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read gradle file: %w", err)
	}

	var deps []Dependency
	matches := gradleDepRe.FindAllStringSubmatch(string(data), -1)
	for _, m := range matches {
		deps = append(deps, Dependency{
			GroupID:    m[1],
			ArtifactID: m[2],
			Version:    m[3],
		})
	}

	return deps, nil
}

// resolveDependencies resolves each dependency to its JAR path via maven-indexer-cli.
func resolveDependencies(deps []Dependency) ([]string, error) {
	var jarPaths []string
	seen := make(map[string]bool)

	for _, dep := range deps {
		query := dep.ArtifactID
		if dep.GroupID != "" {
			query = dep.GroupID + ":" + dep.ArtifactID
		}

		artifacts, err := searchArtifacts(query)
		if err != nil {
			continue
		}

		matched := findBestMatch(artifacts, dep)
		if matched == nil {
			continue
		}

		jarPath := resolveJarPath(matched)
		if jarPath == "" {
			continue
		}

		if !seen[jarPath] {
			seen[jarPath] = true
			jarPaths = append(jarPaths, jarPath)
		}
	}

	return jarPaths, nil
}

// searchArtifacts invokes maven-indexer-cli search-artifacts --json and parses the output.
func searchArtifacts(query string) ([]Artifact, error) {
	cmd := exec.Command(cliName, "search-artifacts", query, "--json", "--limit", "10")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("maven-indexer-cli search-artifacts failed: %w", err)
	}

	var artifacts []Artifact
	if err := json.Unmarshal(output, &artifacts); err != nil {
		return nil, fmt.Errorf("failed to parse maven-indexer-cli output: %w", err)
	}

	return artifacts, nil
}

// findBestMatch selects the best matching artifact for a given dependency.
// It prefers exact groupId+artifactId matches and the requested version.
func findBestMatch(artifacts []Artifact, dep Dependency) *Artifact {
	var groupMatch *Artifact
	var firstMatch *Artifact

	for i := range artifacts {
		a := &artifacts[i]
		if a.GroupID == dep.GroupID && a.ArtifactID == dep.ArtifactID {
			if dep.Version != "" && a.Version == dep.Version {
				return a
			}
			if groupMatch == nil {
				groupMatch = a
			}
		}
		if firstMatch == nil {
			firstMatch = a
		}
	}

	if groupMatch != nil {
		return groupMatch
	}

	return firstMatch
}

// resolveJarPath determines the JAR file path from a resolved artifact.
// It uses the artifact's abspath (the Maven local repo directory) and
// constructs the expected JAR filename.
func resolveJarPath(artifact *Artifact) string {
	if artifact.AbsPath == "" {
		return ""
	}

	jarName := fmt.Sprintf("%s-%s.jar", artifact.ArtifactID, artifact.Version)
	jarPath := filepath.Join(artifact.AbsPath, jarName)

	if _, err := os.Stat(jarPath); err == nil {
		return jarPath
	}

	return artifact.AbsPath
}

// HasBuildFile returns true if sourceDir contains a Maven or Gradle build file.
func HasBuildFile(sourceDir string) bool {
	for _, f := range []string{"pom.xml", "build.gradle", "build.gradle.kts"} {
		if _, err := os.Stat(filepath.Join(sourceDir, f)); err == nil {
			return true
		}
	}
	return false
}

// IsAvailable returns true if maven-indexer-cli is on PATH.
func IsAvailable() bool {
	return isCLIAvailable()
}

// ResolveClass attempts to resolve a class name via maven-indexer-cli get-class.
// Returns the class detail output as a string, or empty string if not found.
func ResolveClass(className string) string {
	if !isCLIAvailable() {
		return ""
	}

	cmd := exec.Command(cliName, "get-class", className, "--json", "--type", "signatures")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// SearchClasses searches for classes matching the query via maven-indexer-cli.
func SearchClasses(query string) ([]Artifact, error) {
	if !isCLIAvailable() {
		return nil, nil
	}

	cmd := exec.Command(cliName, "search-classes", query, "--json", "--limit", "10")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("maven-indexer-cli search-classes failed: %w", err)
	}

	type classResult struct {
		ClassName string     `json:"className"`
		Artifacts []Artifact `json:"artifacts"`
	}

	var results []classResult
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("failed to parse maven-indexer-cli output: %w", err)
	}

	var allArtifacts []Artifact
	for _, r := range results {
		allArtifacts = append(allArtifacts, r.Artifacts...)
	}

	return allArtifacts, nil
}

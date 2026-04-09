// Package maven integrates with maven-indexer-cli to resolve dependency JARs.
// See: https://github.com/tangcent/maven-indexer-cli
package maven

// Resolve attempts to resolve Maven/Gradle dependency coordinates in sourceDir
// by invoking maven-indexer-cli as a subprocess.
// Returns the list of resolved JAR paths for type analysis.
func Resolve(sourceDir string) ([]string, error) {
	// TODO: implement
	// 1. Detect pom.xml or build.gradle in sourceDir
	// 2. Invoke maven-indexer-cli with the project coordinates
	// 3. Return the list of downloaded JAR paths
	return nil, nil
}

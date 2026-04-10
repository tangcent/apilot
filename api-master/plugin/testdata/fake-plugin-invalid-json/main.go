// fake-plugin-invalid-json is a test helper binary used by subprocess_test.go.
// It always outputs invalid JSON to test error handling.
package main

import "fmt"

func main() {
	fmt.Print("invalid json")
}

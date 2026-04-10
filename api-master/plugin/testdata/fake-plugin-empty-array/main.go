// fake-plugin-empty-array is a test helper binary used by subprocess_test.go.
// It always outputs an empty JSON array.
package main

import "fmt"

func main() {
	fmt.Print(`[]`)
}

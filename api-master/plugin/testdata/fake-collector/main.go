// fake-collector is a test helper binary used by subprocess_test.go.
// It simulates a collector subprocess plugin.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--supported-languages" {
		fmt.Print(`["java", "kotlin"]`)
		return
	}
	fmt.Print(`[]`)
}

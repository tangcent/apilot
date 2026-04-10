// fake-plugin-with-args is a test helper binary used by subprocess_test.go.
// It returns ["test"] only when called with --arg1 --supported-languages.
package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]
	if len(args) == 2 && args[0] == "--arg1" && args[1] == "--supported-languages" {
		fmt.Print(`["test"]`)
		return
	}
	fmt.Print(`[]`)
}

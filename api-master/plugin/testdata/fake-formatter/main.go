// fake-formatter is a test helper binary used by subprocess_test.go.
// It simulates a formatter subprocess plugin: reads stdin and writes "ok".
package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	io.ReadAll(os.Stdin) //nolint:errcheck
	fmt.Print("ok")
}

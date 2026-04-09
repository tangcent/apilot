// api-master is the core engine binary.
// It loads plugins from a registry and orchestrates the collect → format → output pipeline.
// For a batteries-included CLI with all collectors/formatters bundled, use apilot-cli instead.
package main

import "github.com/tangcent/apilot/api-master/engine"

func main() {
	engine.RunCLI()
}

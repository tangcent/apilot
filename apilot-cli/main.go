// apilot-cli is the batteries-included CLI binary.
// It statically links all built-in collectors and formatters and delegates to the api-master engine.
// No external plugin registry is required for standard use.
package main

import (
	"github.com/tangcent/apilot/api-master/engine"

	// Collectors
	gocollector   "github.com/tangcent/apilot/api-collector-go"
	javacollector "github.com/tangcent/apilot/api-collector-java"
	nodecollector "github.com/tangcent/apilot/api-collector-node"
	pycollector   "github.com/tangcent/apilot/api-collector-python"

	// Formatters
	curlfmt    "github.com/tangcent/apilot/api-formatter-curl"
	mdfmt      "github.com/tangcent/apilot/api-formatter-markdown"
	postmanfmt "github.com/tangcent/apilot/api-formatter-postman"
)

func init() {
	engine.RegisterCollector(javacollector.New())
	engine.RegisterCollector(gocollector.New())
	engine.RegisterCollector(nodecollector.New())
	engine.RegisterCollector(pycollector.New())

	engine.RegisterFormatter(mdfmt.New())
	engine.RegisterFormatter(curlfmt.New())
	engine.RegisterFormatter(postmanfmt.New())
}

func main() {
	engine.RunCLI()
}

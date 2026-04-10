package main

import (
	"fmt"
	"os"

	"github.com/tangcent/apilot/api-master/engine"
	"github.com/tangcent/apilot/apilot-cli/build"

	gocollector   "github.com/tangcent/apilot/api-collector-go"
	javacollector "github.com/tangcent/apilot/api-collector-java"
	nodecollector "github.com/tangcent/apilot/api-collector-node"
	pycollector   "github.com/tangcent/apilot/api-collector-python"

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
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			fmt.Printf("apilot %s (built %s)\n", build.Version, build.Date)
			os.Exit(0)
		}
	}

	engine.RunCLI()
}

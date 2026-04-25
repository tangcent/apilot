module github.com/tangcent/apilot/api-collector-python

go 1.23

toolchain go1.24.4

require (
	github.com/tangcent/apilot/api-collector v0.0.0
	github.com/tangcent/apilot/api-model v0.0.0
	github.com/tree-sitter/go-tree-sitter v0.25.0
	github.com/tree-sitter/tree-sitter-python v0.23.6
)

require github.com/mattn/go-pointer v0.0.1 // indirect

replace github.com/tangcent/apilot/api-collector => ../api-collector

replace github.com/tangcent/apilot/api-model => ../api-model

module github.com/tangcent/apilot/api-collector-node

go 1.23

toolchain go1.24.4

require github.com/tangcent/apilot/api-collector v0.0.0

require github.com/tangcent/apilot/api-model v0.0.0 // indirect

require (
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/tree-sitter/go-tree-sitter v0.25.0
	github.com/tree-sitter/tree-sitter-javascript v0.25.0
	github.com/tree-sitter/tree-sitter-typescript v0.23.2
)

replace (
	github.com/tangcent/apilot/api-collector => ../api-collector
	github.com/tangcent/apilot/api-model => ../api-model
)

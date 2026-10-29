module github.com/tangcent/apilot/api-collector-kotlin

go 1.23

require (
	github.com/tangcent/apilot/api-collector v0.0.0
	github.com/tangcent/apilot/api-collector-java v0.0.0
	github.com/tangcent/apilot/api-model v0.0.0
	github.com/tree-sitter-grammars/tree-sitter-kotlin v1.1.0
	github.com/tree-sitter/go-tree-sitter v0.25.0
)

require (
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/tangcent/apilot/api-docmeta v0.0.0 // indirect
	github.com/tree-sitter/tree-sitter-java v0.23.5 // indirect
)

replace github.com/tangcent/apilot/api-collector => ../api-collector

replace github.com/tangcent/apilot/api-collector-java => ../api-collector-java

replace github.com/tangcent/apilot/api-docmeta => ../api-docmeta

replace github.com/tangcent/apilot/api-model => ../api-model

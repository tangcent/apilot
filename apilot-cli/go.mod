module github.com/tangcent/apilot/apilot-cli

go 1.22

require (
	github.com/tangcent/apilot/api-collector-go v0.0.0
	github.com/tangcent/apilot/api-collector-java v0.0.0
	github.com/tangcent/apilot/api-collector-node v0.0.0
	github.com/tangcent/apilot/api-collector-python v0.0.0
	github.com/tangcent/apilot/api-formatter-curl v0.0.0
	github.com/tangcent/apilot/api-formatter-markdown v0.0.0
	github.com/tangcent/apilot/api-formatter-postman v0.0.0
	github.com/tangcent/apilot/api-master v0.0.0
)

require (
	github.com/tangcent/apilot/api-collector v0.0.0 // indirect
	github.com/tangcent/apilot/api-formatter v0.0.0 // indirect
)

replace (
	github.com/tangcent/apilot/api-collector => ../api-collector
	github.com/tangcent/apilot/api-collector-go => ../api-collector-go
	github.com/tangcent/apilot/api-collector-java => ../api-collector-java
	github.com/tangcent/apilot/api-collector-node => ../api-collector-node
	github.com/tangcent/apilot/api-collector-python => ../api-collector-python
	github.com/tangcent/apilot/api-formatter => ../api-formatter
	github.com/tangcent/apilot/api-formatter-curl => ../api-formatter-curl
	github.com/tangcent/apilot/api-formatter-markdown => ../api-formatter-markdown
	github.com/tangcent/apilot/api-formatter-postman => ../api-formatter-postman
	github.com/tangcent/apilot/api-master => ../api-master
)

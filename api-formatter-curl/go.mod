module github.com/tangcent/apilot/api-formatter-curl

go 1.22

require (
	github.com/tangcent/apilot/api-collector v0.0.0
	github.com/tangcent/apilot/api-formatter v0.0.0
)

replace (
	github.com/tangcent/apilot/api-collector => ../api-collector
	github.com/tangcent/apilot/api-formatter => ../api-formatter
)
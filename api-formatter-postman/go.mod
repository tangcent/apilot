module github.com/tangcent/apilot/api-formatter-postman

go 1.22

require (
	github.com/tangcent/apilot/api-model v0.0.0
	github.com/tangcent/apilot/api-formatter v0.0.0
)

replace (
	github.com/tangcent/apilot/api-model => ../api-model
	github.com/tangcent/apilot/api-formatter => ../api-formatter
)

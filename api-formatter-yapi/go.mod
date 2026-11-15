module github.com/tangcent/apilot/api-formatter-yapi

go 1.22

require (
	github.com/tangcent/apilot/api-formatter v0.0.0
	github.com/tangcent/apilot/api-model v0.0.0
)

require github.com/yuin/goldmark v1.8.7

replace (
	github.com/tangcent/apilot/api-formatter => ../api-formatter
	github.com/tangcent/apilot/api-model => ../api-model
)

package collector

import "github.com/tangcent/apilot/api-model"

// Re-export model types so existing collector implementations need no import changes.
type ApiEndpoint = model.ApiEndpoint
type ApiParameter = model.ApiParameter
type ApiHeader = model.ApiHeader
type ApiBody = model.ApiBody

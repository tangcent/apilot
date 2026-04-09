// Package jaxrs parses JAX-RS annotated classes for API endpoints.
package jaxrs

import "github.com/tangcent/apilot/api-collector/collector"

// Parse extracts endpoints from JAX-RS annotated source files under sourceDir.
// It looks for @Path, @GET, @POST, @PUT, @DELETE, @Produces, @Consumes.
func Parse(sourceDir string) ([]collector.ApiEndpoint, error) {
	// TODO: implement
	return nil, nil
}

package jaxrs

import (
	"strings"

	"github.com/tangcent/apilot/api-collector-java/expr"
)

// JAX-RS Response is opaque: the declared type names no payload, and the
// payload is built inside the method body (`Response.ok(entity)`). Reading
// only the return type loses the entire response section, because the wrapper
// resolves to nothing at all.
//
// The helpers below recover the entity from the body text. The body is short,
// the shapes are few and regular, and a receiver or argument that cannot be
// named is reported as opaque instead of guessed at.

// isJaxrsResponse reports whether returnType denotes the JAX-RS Response type,
// in either namespace: javax.ws.rs until Jakarta EE 9, jakarta.ws.rs after it.
func isJaxrsResponse(returnType string) bool {
	switch strings.TrimSpace(returnType) {
	case "Response", "javax.ws.rs.core.Response", "jakarta.ws.rs.core.Response":
		return true
	}
	return false
}

// responsePayload recovers the payload type of a JAX-RS Response from the
// method body, where the entity is built rather than declared.
//
// body is the raw method body, params maps the method's parameter names to
// their declared types, and lookup supplies the type-parameter counts that
// turn `Result.success(category)` into `Result<Category>`.
//
// The result is one of three states:
//
//	payload != ""      the entity type, ready for the type resolver
//	payload == "", !opaque  the body builds a Response with no entity
//	                        (`Response.noContent().build()`): nothing is lost
//	payload == "", opaque   a payload was built that this inspection cannot
//	                        name; the caller must say so out loud
func responsePayload(body string, params map[string]string, lookup expr.Lookup) (string, bool) {
	entity, found := responseEntityExpr(body)
	if !found {
		// No Response construction at all: the value is produced elsewhere
		// (a helper call, a variable assigned in several branches).
		return "", true
	}
	if entity == "" {
		// A construction with no entity. The body genuinely has no payload,
		// which is not a gap worth a warning.
		return "", false
	}
	if typeName, ok := expr.Type(entity, params, body, lookup); ok {
		return typeName, false
	}
	return "", true
}

// responseEntityExpr finds the entity a Response is built with: the earliest
// entity-bearing construction in the body, since a method that attaches an
// entity anywhere in its branches is documenting that payload.
//
// found is false when the body contains no recognised Response construction.
// An empty entity with found true means the constructions carry no entity.
func responseEntityExpr(body string) (entity string, found bool) {
	// `.entity(x)` is the explicit builder call and can appear on any
	// ResponseBuilder, including one obtained from Response.status(...).
	if idx := strings.Index(body, ".entity("); idx >= 0 {
		found = true
		if arg, ok := expr.FirstCallArgument(body[idx:], ".entity(", 0); ok && arg != "" {
			return arg, true
		}
	}

	// The factory methods that take an entity. Positions follow the JAX-RS
	// API: ok/accepted/serverError take it first, status/created after their
	// leading status or location argument. A call found without an argument at
	// that position (`ok()`, `status(204)`) is a construction with no entity.
	entities := []struct {
		method string
		index  int
	}{
		{"Response.ok(", 0},
		{"Response.accepted(", 0},
		{"Response.serverError(", 0},
		{"Response.status(", 1},
		{"Response.created(", 1},
	}

	earliest := -1
	for _, candidate := range entities {
		idx := strings.Index(body, candidate.method)
		if idx < 0 {
			continue
		}
		found = true
		arg, ok := expr.FirstCallArgument(body[idx:], candidate.method, candidate.index)
		if !ok || arg == "" {
			continue
		}
		if earliest < 0 || idx < earliest {
			earliest, entity = idx, arg
		}
	}
	if earliest >= 0 {
		return entity, true
	}

	// Factories that never carry an entity. They are recognised so that a
	// `Response.noContent().build()` body reports "no payload" rather than the
	// payload it never had being called unrecoverable.
	for _, call := range []string{
		"Response.noContent(",
		"Response.notModified(",
		"Response.seeOther(",
		"Response.temporaryRedirect(",
	} {
		if strings.Contains(body, call) {
			found = true
		}
	}

	return "", found
}

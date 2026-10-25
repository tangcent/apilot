package jaxrs

import (
	"fmt"
	"testing"

	"github.com/tangcent/apilot/api-collector-java/expr"
	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestIsJaxrsResponse(t *testing.T) {
	responses := []string{"Response", "javax.ws.rs.core.Response", "jakarta.ws.rs.core.Response"}
	for _, returnType := range responses {
		if !isJaxrsResponse(returnType) {
			t.Errorf("isJaxrsResponse(%q) = false, want true", returnType)
		}
	}

	// A project's own Response<T> is a class of its own and must reach the
	// resolver instead of being read as the JAX-RS wrapper.
	others := []string{"", "ResponseEntity<Order>", "Responses", "HttpResponse"}
	for _, returnType := range others {
		if isJaxrsResponse(returnType) {
			t.Errorf("isJaxrsResponse(%q) = true, want false", returnType)
		}
	}
}

// lookupOf builds a type lookup from a class name to type parameter count map.
// Only the count matters to body inspection.
func lookupOf(counts map[string]int) expr.Lookup {
	registry := make(map[string]parser.Class, len(counts))
	for name, count := range counts {
		params := make([]string, count)
		for i := range params {
			params[i] = fmt.Sprintf("T%d", i)
		}
		registry[name] = parser.Class{Name: name, TypeParameters: params}
	}
	return func(name string) (parser.Class, bool) {
		class, found := registry[name]
		return class, found
	}
}

func TestResponsePayload(t *testing.T) {
	params := map[string]string{"order": "Order", "id": "Long"}
	registry := lookupOf(map[string]int{"Result": 1, "PageResult": 1})

	tests := []struct {
		name       string
		body       string
		want       string
		wantOpaque bool
	}{
		{
			name: "constructor argument",
			body: "{ return Response.ok(new Category()).build(); }",
			want: "Category",
		},
		{
			name: "method parameter",
			body: "{ return Response.ok(order).build(); }",
			want: "Order",
		},
		{
			name: "local variable",
			body: `{
				Category category = new Category();
				return Response.ok(category).build();
			}`,
			want: "Category",
		},
		{
			name: "local variable with type arguments",
			body: `{
				PageResult<Order> page = new PageResult<>();
				return Response.ok(Result.success(page)).build();
			}`,
			want: "Result<PageResult<Order>>",
		},
		{
			name: "generic wrapper takes the argument type",
			body: `{
				Order order = load();
				return Response.ok(Result.success(order)).build();
			}`,
			want: "Result<Order>",
		},
		{
			name: "diamond constructor keeps the raw class",
			body: "{ return Response.ok(Result.success(new PageResult<>())).build(); }",
			want: "Result<PageResult>",
		},
		{
			name: "explicit type argument on the constructor",
			body: "{ return Response.ok(new PageResult<Order>()).build(); }",
			want: "PageResult<Order>",
		},
		{
			name: "builder entity call",
			body: `{
				Order order = new Order();
				return Response.status(201).entity(Result.success(order)).build();
			}`,
			want: "Result<Order>",
		},
		{
			name: "entity call on a branched builder",
			body: `{
				ResponseBuilder builder = Response.status(200);
				builder.entity(order);
				return builder.build();
			}`,
			want: "Order",
		},
		{
			name: "multi-line chain",
			body: `{
				Order order = new Order();
				return Response
					.status(201)
					.entity(Result.success(order))
					.build();
			}`,
			want: "Result<Order>",
		},
		{
			name: "no entity",
			body: "{ return Response.noContent().build(); }",
		},
		{
			name: "empty builder",
			body: "{ return Response.ok().build(); }",
		},
		{
			name: "status only",
			body: "{ return Response.status(204).build(); }",
		},
		{
			name:       "payload from an instance method",
			body:       "{ return Response.ok(service.load(id)).build(); }",
			wantOpaque: true,
		},
		{
			name:       "payload from a local method",
			body:       "{ return Response.ok(load(id)).build(); }",
			wantOpaque: true,
		},
		{
			name: "response built by another method",
			body: `{
				Response response = mapper.toResponse(entity);
				return response;
			}`,
			wantOpaque: true,
		},
		{
			name: "branches that attach no entity",
			body: `{
				Response response;
				if (id == null) {
					response = Response.status(400).build();
				} else {
					response = Response.status(200).build();
				}
				return response;
			}`,
		},
		{
			name:       "no construction at all",
			body:       "{ return delegate.remove(id); }",
			wantOpaque: true,
		},
		{
			name:       "declaration without a body",
			body:       "",
			wantOpaque: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, opaque := responsePayload(tt.body, params, registry)
			if got != tt.want || opaque != tt.wantOpaque {
				t.Errorf("responsePayload() = (%q, %v), want (%q, %v)", got, opaque, tt.want, tt.wantOpaque)
			}
		})
	}
}

func TestResponsePayload_UnknownClassArity(t *testing.T) {
	// A class the parser could not see has no known type parameters, so the
	// wrapper is kept as written rather than guessed at.
	got, opaque := responsePayload(
		"{ Order order = new Order(); return Response.ok(Factory.wrap(order)).build(); }",
		nil,
		lookupOf(nil),
	)
	if got != "Factory" || opaque {
		t.Errorf("responsePayload() = (%q, %v), want (%q, false)", got, opaque, "Factory")
	}
}

package expr

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// lookupOf builds a type lookup from a class name to type parameter count map.
// Only the count matters to body inspection.
func lookupOf(counts map[string]int) Lookup {
	registry := make(map[string]parser.Class, len(counts))
	for name, count := range counts {
		params := make([]string, count)
		for i := range params {
			params[i] = "T"
		}
		registry[name] = parser.Class{Name: name, TypeParameters: params}
	}
	return func(name string) (parser.Class, bool) {
		class, found := registry[name]
		return class, found
	}
}

func TestLocalVariableTypeIgnoresComparisonsAndAssignments(t *testing.T) {
	body := `{
		if (order == null) {
			this.order = order;
		}
		Order order = new Order();
		return Response.ok(order).build();
	}`

	got, ok := localVariableType(body, "order")
	if !ok || got != "Order" {
		t.Errorf("localVariableType() = (%q, %v), want (\"Order\", true)", got, ok)
	}
}

// TestReturnedType covers the source evidence a wildcard return type is filled
// from: a constructor call, a local variable, a parameter, or a static factory
// whose type argument comes from the payload it wraps (issue #167).
func TestReturnedType(t *testing.T) {
	lookup := lookupOf(map[string]int{"Result": 1, "PageResult": 1})

	tests := []struct {
		name string
		body string
		// params declares the method parameters, as the parser supplies them.
		params map[string]string
		want   string
	}{
		{
			name: "diamond constructor under a factory",
			body: "{ return Result.success(new PageResult<>()); }",
			want: "Result<PageResult>",
		},
		{
			name: "local variable with type arguments",
			body: `{
				PageResult<Order> page = new PageResult<>();
				return Result.success(page);
			}`,
			want: "Result<PageResult<Order>>",
		},
		{
			name:   "method parameter",
			body:   "{ return Result.success(page); }",
			params: map[string]string{"page": "PageResult"},
			want:   "Result<PageResult>",
		},
		{
			// The wrapper is named, its type argument is not: the raw Result
			// cannot fill a wildcard, so the caller keeps its open argument.
			name: "anonymous value from an instance call",
			body: "{ return Result.success(service.pageOrders(request)); }",
			want: "Result",
		},
		{
			name: "conflicting branches",
			body: `{
				if (id == null) {
					return Result.success(new PageResult<>());
				}
				return Result.success(new Order());
			}`,
			want: "",
		},
		{
			name: "no return at all",
			body: "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method := parser.Method{Body: tt.body}
			for name, typeName := range tt.params {
				method.Parameters = append(method.Parameters, parser.Parameter{Name: name, Type: typeName})
			}

			got, ok := ReturnedType(method, lookup)
			if got != tt.want {
				t.Errorf("ReturnedType() = (%q, %v), want (%q, %v)", got, ok, tt.want, tt.want != "")
			}
		})
	}
}

// TestReturnedTypeSkipsNullReturns keeps a `return null` branch from being read
// as conflicting evidence: it names no type, so the other branch still decides.
func TestReturnedTypeSkipsNullReturns(t *testing.T) {
	body := `{
		if (id == null) {
			return null;
		}
		return Result.success(new Order());
	}`

	got, ok := ReturnedType(parser.Method{Body: body}, lookupOf(map[string]int{"Result": 1}))
	if !ok || got != "Result<Order>" {
		t.Errorf("ReturnedType() = (%q, %v), want (\"Result<Order>\", true)", got, ok)
	}
}

package nestjs

import (
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

func TestUnwrapPromise(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Promise<UserResponse>", "UserResponse"},
		{"Promise<void>", "void"},
		{"Promise<Array<UserResponse>>", "Array<UserResponse>"},
		{"UserResponse", "UserResponse"},
		{"string", "string"},
		{"  Promise<UserResponse>  ", "UserResponse"},
	}

	for _, tt := range tests {
		result := unwrapPromise(tt.input)
		if result != tt.expected {
			t.Errorf("unwrapPromise(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestExtractNestJSDecoratorName(t *testing.T) {
	tests := []struct {
		source       string
		decoratorIdx int
		expected     string
	}{
		{"class C { m(@Body() body: string) {} }", 0, "Body"},
		{"class C { m(@Param('id') id: string) {} }", 0, "Param"},
		{"class C { m(@Query('page') page: number) {} }", 0, "Query"},
		{"class C { m(@Headers('x-custom') h: string) {} }", 0, "Headers"},
	}

	for _, tt := range tests {
		p := tree_sitter.NewParser()
		lang := tree_sitter.NewLanguage(typescript.LanguageTypescript())
		p.SetLanguage(lang)

		tree := p.Parse([]byte(tt.source), nil)
		if tree == nil {
			p.Close()
			t.Fatalf("failed to parse: %s", tt.source)
		}
		root := tree.RootNode()

		var decoratorNode *tree_sitter.Node
		var walk func(n *tree_sitter.Node)
		walk = func(n *tree_sitter.Node) {
			if decoratorNode != nil {
				return
			}
			if n.Kind() == "decorator" {
				decoratorNode = n
				return
			}
			for i := uint(0); i < n.ChildCount(); i++ {
				walk(n.Child(i))
			}
		}
		walk(root)

		if decoratorNode == nil {
			tree.Close()
			p.Close()
			t.Fatalf("no decorator found in: %s", tt.source)
		}
		result := extractNestJSDecoratorName(decoratorNode, []byte(tt.source))
		if result != tt.expected {
			t.Errorf("extractNestJSDecoratorName(%q) = %q, want %q", tt.source, result, tt.expected)
		}
		tree.Close()
		p.Close()
	}
}

func TestPickBestApiResponse(t *testing.T) {
	responses := []ApiResponseInfo{
		{Status: "400", Description: "Bad request", TypeName: "ErrorResponse"},
		{Status: "200", Description: "Success", TypeName: "UserResponse"},
		{Status: "500", Description: "Server error", TypeName: "ErrorResponse"},
	}

	best := pickBestApiResponse(responses)
	if best == nil {
		t.Fatal("expected non-nil response")
	}
	if best.Status != "200" {
		t.Errorf("expected status 200, got %s", best.Status)
	}
	if best.TypeName != "UserResponse" {
		t.Errorf("expected UserResponse, got %s", best.TypeName)
	}
}

func TestPickBestApiResponse_201(t *testing.T) {
	responses := []ApiResponseInfo{
		{Status: "201", Description: "Created", TypeName: "UserResponse"},
		{Status: "400", Description: "Bad request", TypeName: "ErrorResponse"},
	}

	best := pickBestApiResponse(responses)
	if best == nil {
		t.Fatal("expected non-nil response")
	}
	if best.Status != "201" {
		t.Errorf("expected status 201, got %s", best.Status)
	}
}

func TestPickBestApiResponse_NoSuccess(t *testing.T) {
	responses := []ApiResponseInfo{
		{Status: "400", Description: "Bad request", TypeName: "ErrorResponse"},
	}

	best := pickBestApiResponse(responses)
	if best == nil {
		t.Fatal("expected non-nil response (fallback)")
	}
	if best.TypeName != "ErrorResponse" {
		t.Errorf("expected ErrorResponse as fallback, got %s", best.TypeName)
	}
}

func TestPickBestApiResponse_Empty(t *testing.T) {
	responses := []ApiResponseInfo{}
	best := pickBestApiResponse(responses)
	if best != nil {
		t.Errorf("expected nil for empty responses, got %+v", best)
	}
}

package fastify

import (
	"path/filepath"
	"testing"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"

	collector "github.com/tangcent/apilot/api-collector"
)

func TestExtractSchemaQuery(t *testing.T) {
	withSchemaNode(t, `fastify.get('/search', { schema: { querystring: {
		type: 'object',
		required: ['keyword'],
		properties: { keyword: { type: 'string' }, limit: { type: 'integer' } }
	} } }, search);`, func(schema *tree_sitter.Node, src []byte) {
		got := extractSchemaQuery(schema, src, collector.NewUnresolvedSet())

		if got == nil || !got.IsObject() {
			t.Fatalf("extractSchemaQuery = %+v, want object", got)
		}
		keyword, ok := got.Fields["keyword"]
		if !ok {
			t.Fatal("querystring should have 'keyword'")
		}
		if !keyword.Required {
			t.Errorf("'keyword' should be required")
		}
		if limit := got.Fields["limit"]; limit == nil || limit.Required {
			t.Errorf("'limit' should be present and optional, got %+v", limit)
		}
	})
}

func TestExtractSchemaParams(t *testing.T) {
	withSchemaNode(t, `fastify.get('/users/:id', { schema: { params: {
		type: 'object',
		properties: { id: { type: 'string' } }
	} } }, getUser);`, func(schema *tree_sitter.Node, src []byte) {
		got := extractSchemaParams(schema, src, collector.NewUnresolvedSet())

		if got == nil || !got.IsObject() {
			t.Fatalf("extractSchemaParams = %+v, want object", got)
		}
		if _, ok := got.Fields["id"]; !ok {
			t.Errorf("params should have 'id', got %v", got.Fields)
		}
	})
}

// A response object without 200/201/default falls back to the first status
// that yields a schema.
func TestExtractSchemaResponseFallsBackToFirstStatus(t *testing.T) {
	withSchemaNode(t, `fastify.get('/ping', { schema: { response: {
		404: { type: 'object', properties: { msg: { type: 'string' } } }
	} } }, ping);`, func(schema *tree_sitter.Node, src []byte) {
		got := extractSchemaResponse(schema, src, collector.NewUnresolvedSet())

		if got == nil || !got.IsObject() {
			t.Fatalf("extractSchemaResponse = %+v, want object from the 404 branch", got)
		}
		if _, ok := got.Fields["msg"]; !ok {
			t.Errorf("response should have 'msg', got %v", got.Fields)
		}
	})
}

func TestExtractSchemaBodyArrayWithItems(t *testing.T) {
	withSchemaNode(t, `fastify.post('/batch', { schema: { body: {
		type: 'array',
		items: { type: 'object', properties: { id: { type: 'integer' } } }
	} } }, batch);`, func(schema *tree_sitter.Node, src []byte) {
		got := extractSchemaBody(schema, src, collector.NewUnresolvedSet())

		if got == nil || !got.IsArray() {
			t.Fatalf("extractSchemaBody = %+v, want array", got)
		}
		if !got.Items.IsObject() {
			t.Fatalf("items kind = %s, want object", got.Items.Kind)
		}
		if _, ok := got.Items.Fields["id"]; !ok {
			t.Errorf("items should have 'id', got %v", got.Items.Fields)
		}
	})
}

func TestMapJSONSchemaTypeUnknownIsRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()

	got := mapJSONSchemaType("CustomPage", sink)

	if !got.IsSingle() || got.TypeName != "CustomPage" {
		t.Fatalf("mapJSONSchemaType(CustomPage) = (%s, %q), want single CustomPage", got.Kind, got.TypeName)
	}
	if sink.Counts()["CustomPage"] != 1 {
		t.Errorf("unresolved counts = %v, want CustomPage:1", sink.Counts())
	}

	mapJSONSchemaType("string", sink)
	if sink.Counts()["string"] != 0 {
		t.Errorf("known JSON-schema types are not failures, got %v", sink.Counts())
	}
}

// A schema written as a bare identifier cannot be expanded from the JSON
// schema itself; the parser resolves it through the type registry and records
// the name when the registry does not know it.
func TestParseWithUnresolved_IdentifierSchemaIsRecorded(t *testing.T) {
	sink := collector.NewUnresolvedSet()

	endpoints, err := ParseWithUnresolved(filepath.Join("testdata", "unresolved"), nil, sink)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if len(endpoints) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(endpoints))
	}
	ep := endpoints[0]

	if ep.RequestBody == nil || !ep.RequestBody.Body.IsSingle() {
		t.Fatalf("request body = %+v, want single model", ep.RequestBody)
	}
	if ep.RequestBody.Body.TypeName != "echoRequest" {
		t.Errorf("request body type = %q, want echoRequest", ep.RequestBody.Body.TypeName)
	}
	if ep.Response == nil || !ep.Response.Body.IsSingle() {
		t.Fatalf("response = %+v, want single model", ep.Response)
	}
	if ep.Response.Body.TypeName != "echoReply" {
		t.Errorf("response type = %q, want echoReply", ep.Response.Body.TypeName)
	}

	counts := sink.Counts()
	if counts["echoRequest"] != 1 {
		t.Errorf("unresolved counts = %v, want echoRequest:1", counts)
	}
	if counts["echoReply"] != 1 {
		t.Errorf("unresolved counts = %v, want echoReply:1", counts)
	}
}

// withSchemaNode parses a single fastify route statement and hands the route's
// schema object node to fn, mirroring the navigation used by
// extractShorthandRoute. The tree stays alive for the whole callback; nodes
// must not outlive it.
func withSchemaNode(t *testing.T, source string, fn func(*tree_sitter.Node, []byte)) {
	t.Helper()

	src := []byte(source)
	p := tree_sitter.NewParser()
	defer p.Close()
	lang := tree_sitter.NewLanguage(javascript.Language())
	if err := p.SetLanguage(lang); err != nil {
		t.Fatalf("set language: %v", err)
	}
	tree := p.Parse(src, nil)
	if tree == nil {
		t.Fatal("parse returned nil tree")
	}
	defer tree.Close()

	call := firstCallExpression(tree.RootNode())
	if call == nil {
		t.Fatal("no call_expression in source")
	}
	options := findOptionsObject(findArgsNode(call, src), src)
	if options == nil {
		t.Fatal("no options object in source")
	}
	schema := extractSchemaFromOptions(options, src)
	if schema == nil {
		t.Fatal("no schema object in source")
	}
	fn(schema, src)
}

func firstCallExpression(n *tree_sitter.Node) *tree_sitter.Node {
	if n.Kind() == "call_expression" {
		return n
	}
	for i := uint(0); i < n.ChildCount(); i++ {
		if c := firstCallExpression(n.Child(i)); c != nil {
			return c
		}
	}
	return nil
}

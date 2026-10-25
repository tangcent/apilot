package javacollector

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	model "github.com/tangcent/apilot/api-model"
)

func TestCollect_SpringMVC(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints from Spring MVC controller")
	}

	found := false
	for _, ep := range endpoints {
		if ep.Folder == "UserController" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected endpoints from UserController")
	}
}

func TestCollect_JAXRS(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"jaxrs"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints from JAX-RS resource")
	}

	found := false
	for _, ep := range endpoints {
		if ep.Folder == "UserResource" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected endpoints from UserResource")
	}
}

func TestCollect_Feign(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"feign"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	if len(endpoints) == 0 {
		t.Fatal("Expected endpoints from Feign client")
	}

	found := false
	for _, ep := range endpoints {
		if ep.Folder == "UserClient" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected endpoints from UserClient")
	}
}

func TestCollect_AllFrameworks(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	// No framework hints = detect all
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: testdataDir,
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Should find endpoints from Spring MVC (5) + JAX-RS (5) + Feign (4) = 14
	if len(endpoints) < 14 {
		t.Errorf("Expected at least 14 endpoints, got %d", len(endpoints))
	}

	protocols := make(map[string]bool)
	for _, ep := range endpoints {
		protocols[ep.Protocol] = true
	}
	if !protocols["http"] {
		t.Error("Expected http protocol endpoints")
	}
}

func TestCollect_EndpointFields(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	for _, ep := range endpoints {
		if ep.Protocol != "http" {
			t.Errorf("Expected protocol 'http', got '%s'", ep.Protocol)
		}
		if ep.Method == "" {
			t.Errorf("Expected non-empty method for endpoint %s", ep.Name)
		}
	}
}

func TestCollect_FrameworkAliases(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	for _, alias := range []string{"spring", "spring-mvc", "springmvc"} {
		endpoints, err := c.Collect(collector.CollectContext{
			SourceDir:  testdataDir,
			Frameworks: []string{alias},
		})
		if err != nil {
			t.Fatalf("Collect with alias '%s' failed: %v", alias, err)
		}
		if len(endpoints) == 0 {
			t.Errorf("Expected endpoints for alias '%s'", alias)
		}
	}
}

func TestCollect_SpringMVCDocumentation(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	var documented *collector.ApiEndpoint
	for i := range endpoints {
		if endpoints[i].Name == "getDocumented" {
			documented = &endpoints[i]
			break
		}
	}
	if documented == nil {
		t.Fatal("Expected getDocumented endpoint")
	}

	if documented.Description != "Fetch documented item\n\nReturns a documented item by id" {
		t.Fatalf("Expected endpoint description, got %q", documented.Description)
	}

	params := make(map[string]collector.ApiParameter)
	for _, p := range documented.Parameters {
		params[p.Name] = p
	}

	id := params["id"]
	if id.Description != "Documented item id" {
		t.Errorf("Expected id description, got %q", id.Description)
	}
	if id.Example != "42" {
		t.Errorf("Expected id example, got %q", id.Example)
	}
	if !id.Required {
		t.Error("Expected id to be required")
	}

	status := params["status"]
	if status.Description != "fallback status" {
		t.Errorf("Expected status JavaDoc fallback, got %q", status.Description)
	}
	if status.Default != "active" {
		t.Errorf("Expected status default, got %q", status.Default)
	}
	if status.Required {
		t.Error("Expected status to be optional because it has a default")
	}

	if documented.RequestBody == nil || documented.RequestBody.Body == nil {
		t.Fatal("Expected documented request body")
	}
	requestName := documented.RequestBody.Body.Fields["name"]
	if requestName == nil {
		t.Fatal("Expected documented request field name")
	}
	if requestName.Comment != "Requested name" {
		t.Errorf("Expected request field comment, got %q", requestName.Comment)
	}
	if requestName.Demo != "Ada" {
		t.Errorf("Expected request field demo, got %q", requestName.Demo)
	}
	if !requestName.Required {
		t.Error("Expected request field to be required")
	}

	if documented.Response == nil || documented.Response.Body == nil {
		t.Fatal("Expected documented response body")
	}
	responseDisplayName := documented.Response.Body.Fields["displayName"]
	if responseDisplayName == nil {
		t.Fatal("Expected documented response field displayName")
	}
	if responseDisplayName.Comment != "Display name" {
		t.Errorf("Expected response field comment, got %q", responseDisplayName.Comment)
	}
	if responseDisplayName.Demo != "Ada" {
		t.Errorf("Expected response field demo, got %q", responseDisplayName.Demo)
	}
}

func TestCollect_JAXRSDocumentation(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"jaxrs"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	documented := findEndpoint(t, endpoints, "getDocumented")
	if documented.Description != "Fetch documented item\n\nReturns a documented item by id" {
		t.Fatalf("Expected endpoint description, got %q", documented.Description)
	}

	id := findParameter(t, documented, "id")
	if id.Description != "Documented item id" {
		t.Errorf("Expected id description, got %q", id.Description)
	}
	if id.Example != "42" {
		t.Errorf("Expected id example, got %q", id.Example)
	}
	if !id.Required {
		t.Error("Expected id to be required")
	}

	status := findParameter(t, documented, "status")
	if status.Description != "fallback status" {
		t.Errorf("Expected status JavaDoc fallback, got %q", status.Description)
	}
	if status.Default != "active" {
		t.Errorf("Expected status default from @DefaultValue, got %q", status.Default)
	}
	if status.Required {
		t.Error("Expected status to be optional because it has a default")
	}

	// JavaDoc-only endpoint: description and param docs both fall back to JavaDoc.
	search := findEndpoint(t, endpoints, "searchDocumented")
	if search.Description != "Search documented items." {
		t.Errorf("Expected JavaDoc endpoint description, got %q", search.Description)
	}
	keyword := findParameter(t, search, "keyword")
	if keyword.Description != "search keyword" {
		t.Errorf("Expected keyword JavaDoc param fallback, got %q", keyword.Description)
	}
	limit := findParameter(t, search, "limit")
	if limit.Default != "10" {
		t.Errorf("Expected limit default from @DefaultValue, got %q", limit.Default)
	}
}

func TestCollect_FeignDocumentation(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"feign"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	documented := findEndpoint(t, endpoints, "getDocumented")
	if documented.Description != "Fetch documented item." {
		t.Fatalf("Expected endpoint description, got %q", documented.Description)
	}

	id := findParameter(t, documented, "id")
	if id.Description != "documented item id" {
		t.Errorf("Expected id description from JavaDoc, got %q", id.Description)
	}
	if !id.Required {
		t.Error("Expected id to be required")
	}

	status := findParameter(t, documented, "status")
	if status.Description != "fallback status" {
		t.Errorf("Expected status JavaDoc fallback, got %q", status.Description)
	}
	if status.Default != "active" {
		t.Errorf("Expected status default from @RequestParam, got %q", status.Default)
	}

	// Netflix @RequestLine style uses @Param for binding and JavaDoc for docs.
	search := findEndpoint(t, endpoints, "searchDocumented")
	if search.Description != "Search documented items." {
		t.Errorf("Expected JavaDoc endpoint description, got %q", search.Description)
	}
	keyword := findParameter(t, search, "keyword")
	if keyword.Name != "keyword" {
		t.Errorf("Expected @Param alias to name the parameter, got %q", keyword.Name)
	}
	if keyword.Description != "search keyword" {
		t.Errorf("Expected keyword JavaDoc param fallback, got %q", keyword.Description)
	}
}

func findEndpoint(t *testing.T, endpoints []collector.ApiEndpoint, name string) *collector.ApiEndpoint {
	t.Helper()
	for i := range endpoints {
		if endpoints[i].Name == name {
			return &endpoints[i]
		}
	}
	t.Fatalf("Expected endpoint %q", name)
	return nil
}

func findParameter(t *testing.T, endpoint *collector.ApiEndpoint, name string) collector.ApiParameter {
	t.Helper()
	for _, p := range endpoint.Parameters {
		if p.Name == name {
			return p
		}
	}
	t.Fatalf("Expected parameter %q on endpoint %q", name, endpoint.Name)
	return collector.ApiParameter{}
}

func TestCollect_SchemaResolution_SimpleController(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	var healthEp *collector.ApiEndpoint
	for i := range endpoints {
		if endpoints[i].Name == "healthCheck" {
			healthEp = &endpoints[i]
			break
		}
	}
	if healthEp == nil {
		t.Fatal("Expected healthCheck endpoint from BaseController")
	}

	if healthEp.Response == nil || healthEp.Response.Body == nil {
		t.Fatal("Expected ResponseSchema for healthCheck (returns String)")
	}
	if !healthEp.Response.Body.IsSingle() {
		t.Errorf("Expected single model for String return, got kind=%s", healthEp.Response.Body.Kind)
	}
}

func TestCollect_SchemaResolution_OrderController(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	var searchEp *collector.ApiEndpoint
	for i := range endpoints {
		if endpoints[i].Name == "searchByName" {
			searchEp = &endpoints[i]
			break
		}
	}
	if searchEp == nil {
		t.Fatal("Expected searchByName endpoint from OrderController")
	}

	if searchEp.Response == nil || searchEp.Response.Body == nil {
		t.Fatal("Expected ResponseSchema for searchByName (returns OrderVO)")
	}

	resp := searchEp.Response.Body
	if !resp.IsObject() {
		t.Fatalf("Expected object model for OrderVO, got kind=%s", resp.Kind)
	}

	expectedFields := []struct {
		name     string
		kind     model.ObjectModelKind
		typeName string
	}{
		{"id", model.KindSingle, model.JsonTypeLong},
		{"orderId", model.KindSingle, model.JsonTypeString},
		{"customerName", model.KindSingle, model.JsonTypeString},
		{"total", model.KindSingle, model.JsonTypeDouble},
		{"tags", model.KindArray, "array"},
		{"attributes", model.KindMap, "map"},
	}

	for _, ef := range expectedFields {
		field, ok := resp.Fields[ef.name]
		if !ok {
			t.Errorf("Expected field '%s' in OrderVO", ef.name)
			continue
		}
		if field.Model == nil {
			t.Errorf("Field '%s' has nil model", ef.name)
			continue
		}
		if field.Model.Kind != ef.kind {
			t.Errorf("Field '%s': expected kind %s, got %s", ef.name, ef.kind, field.Model.Kind)
		}
		if field.Model.TypeName != ef.typeName {
			t.Errorf("Field '%s': expected typeName %s, got %s", ef.name, ef.typeName, field.Model.TypeName)
		}
	}

	tagsField := resp.Fields["tags"]
	if tagsField != nil && tagsField.Model != nil && tagsField.Model.Items != nil {
		if tagsField.Model.Items.TypeName != model.JsonTypeString {
			t.Errorf("Expected tags items to be string, got %s", tagsField.Model.Items.TypeName)
		}
	}

	// `Map<String, Object>`: the value is java.lang.Object, which carries no
	// fields, so an empty object is its true shape rather than an opaque scalar.
	attrsField := resp.Fields["attributes"]
	if attrsField != nil && attrsField.Model != nil && attrsField.Model.ValueModel != nil {
		if !attrsField.Model.ValueModel.IsObject() {
			t.Errorf("Expected attributes value model to be an empty object, got kind=%s", attrsField.Model.ValueModel.Kind)
		}
	}
}

func TestCollect_SchemaResolution_InheritedEndpoints(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	orderEndpoints := make(map[string]*collector.ApiEndpoint)
	for i := range endpoints {
		if endpoints[i].Folder == "OrderController" {
			ep := endpoints[i]
			orderEndpoints[ep.Name] = &ep
		}
	}

	createEp, ok := orderEndpoints["create"]
	if !ok {
		t.Fatal("Expected inherited 'create' endpoint from BaseCrudController in OrderController")
	}

	if createEp.RequestBody == nil || createEp.RequestBody.Body == nil {
		t.Fatal("Expected RequestBody schema for create endpoint")
	}

	reqBody := createEp.RequestBody.Body
	if !reqBody.IsObject() {
		t.Fatalf("Expected object model for CreateOrderReq, got kind=%s", reqBody.Kind)
	}

	expectedReqFields := []struct {
		name     string
		kind     model.ObjectModelKind
		typeName string
	}{
		{"orderId", model.KindSingle, model.JsonTypeString},
		{"customerName", model.KindSingle, model.JsonTypeString},
		{"amount", model.KindSingle, model.JsonTypeDouble},
		{"items", model.KindArray, "array"},
		{"metadata", model.KindMap, "map"},
	}

	for _, ef := range expectedReqFields {
		field, ok := reqBody.Fields[ef.name]
		if !ok {
			t.Errorf("Expected field '%s' in CreateOrderReq", ef.name)
			continue
		}
		if field.Model == nil {
			t.Errorf("Field '%s' has nil model", ef.name)
			continue
		}
		if field.Model.Kind != ef.kind {
			t.Errorf("Field '%s': expected kind %s, got %s", ef.name, ef.kind, field.Model.Kind)
		}
		if field.Model.TypeName != ef.typeName {
			t.Errorf("Field '%s': expected typeName %s, got %s", ef.name, ef.typeName, field.Model.TypeName)
		}
	}

	if createEp.Response == nil || createEp.Response.Body == nil {
		t.Fatal("Expected Response schema for create endpoint")
	}

	respBody := createEp.Response.Body
	if !respBody.IsObject() {
		t.Fatalf("Expected object model for Result<OrderVO>, got kind=%s", respBody.Kind)
	}

	dataField, ok := respBody.Fields["data"]
	if !ok {
		t.Fatal("Expected 'data' field in Result<OrderVO>")
	}
	if dataField.Model == nil {
		t.Fatal("Expected 'data' field to have a model")
	}
	if !dataField.Model.IsObject() {
		t.Errorf("Expected 'data' field to be object (OrderVO), got kind=%s", dataField.Model.Kind)
	}

	getByIdEp, ok := orderEndpoints["getById"]
	if !ok {
		t.Fatal("Expected inherited 'getById' endpoint from BaseCrudController in OrderController")
	}
	if getByIdEp.Response == nil || getByIdEp.Response.Body == nil {
		t.Fatal("Expected Response schema for getById endpoint")
	}

	listEp, ok := orderEndpoints["list"]
	if !ok {
		t.Fatal("Expected inherited 'list' endpoint from BaseCrudController in OrderController")
	}
	if listEp.Response == nil || listEp.Response.Body == nil {
		t.Fatal("Expected Response schema for list endpoint")
	}

	listResp := listEp.Response.Body
	if !listResp.IsObject() {
		t.Fatalf("Expected object model for PageResult<OrderVO>, got kind=%s", listResp.Kind)
	}

	itemsField, ok := listResp.Fields["items"]
	if !ok {
		t.Fatal("Expected 'items' field in PageResult<OrderVO>")
	}
	if itemsField.Model == nil {
		t.Fatal("Expected 'items' field to have a model")
	}
	if !itemsField.Model.IsArray() {
		t.Errorf("Expected 'items' field to be array, got kind=%s", itemsField.Model.Kind)
	}
}

func TestCollect_SchemaResolution_GenericBaseController(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	var infoEp *collector.ApiEndpoint
	for i := range endpoints {
		if endpoints[i].Name == "getInfo" {
			infoEp = &endpoints[i]
			break
		}
	}
	if infoEp == nil {
		t.Fatal("Expected getInfo endpoint from GenericBaseController")
	}

	if infoEp.Response == nil || infoEp.Response.Body == nil {
		t.Fatal("Expected ResponseSchema for getInfo (returns Result<R>)")
	}

	resp := infoEp.Response.Body
	if !resp.IsObject() {
		t.Fatalf("Expected object model for Result<R>, got kind=%s", resp.Kind)
	}

	dataField, ok := resp.Fields["data"]
	if !ok {
		t.Fatal("Expected 'data' field in Result<R>")
	}
	if dataField.Model == nil {
		t.Fatal("Expected 'data' field to have a model")
	}
	if !dataField.Generic {
		t.Error("Expected 'data' field to be marked as Generic since R is unbound")
	}
}

func TestCollect_SchemaResolution_InheritedModelFields(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	var getUserEp *collector.ApiEndpoint
	for i := range endpoints {
		if endpoints[i].Name == "getUser" && endpoints[i].Folder == "UserController" {
			getUserEp = &endpoints[i]
			break
		}
	}
	if getUserEp == nil {
		t.Fatal("Expected getUser endpoint from UserController")
	}

	if getUserEp.Response == nil || getUserEp.Response.Body == nil {
		t.Fatal("Expected Response body for getUser (returns ResponseEntity<User>)")
	}

	resp := getUserEp.Response.Body
	if !resp.IsObject() {
		t.Fatalf("Expected object model for User, got kind=%s", resp.Kind)
	}

	if resp.TypeName != "User" {
		t.Errorf("Expected typeName 'User', got '%s'", resp.TypeName)
	}

	expectedOwnFields := []struct {
		name     string
		kind     model.ObjectModelKind
		typeName string
	}{
		{"name", model.KindSingle, model.JsonTypeString},
		{"email", model.KindSingle, model.JsonTypeString},
		{"active", model.KindSingle, model.JsonTypeBoolean},
	}

	for _, ef := range expectedOwnFields {
		field, ok := resp.Fields[ef.name]
		if !ok {
			t.Errorf("Expected field '%s' in User", ef.name)
			continue
		}
		if field.Model == nil {
			t.Errorf("Field '%s' has nil model", ef.name)
			continue
		}
		if field.Model.Kind != ef.kind {
			t.Errorf("Field '%s': expected kind %s, got %s", ef.name, ef.kind, field.Model.Kind)
		}
		if field.Model.TypeName != ef.typeName {
			t.Errorf("Field '%s': expected typeName %s, got %s", ef.name, ef.typeName, field.Model.TypeName)
		}
	}

	expectedInheritedFields := []struct {
		name     string
		kind     model.ObjectModelKind
		typeName string
	}{
		{"id", model.KindSingle, model.JsonTypeLong},
		// java.time.LocalDateTime is a scalar on the wire (easy-yapi declares
		// the same `json.rule.convert[java.time.LocalDateTime]=java.lang.String`).
		{"createdAt", model.KindSingle, model.JsonTypeString},
		{"updatedAt", model.KindSingle, model.JsonTypeString},
	}

	for _, ef := range expectedInheritedFields {
		field, ok := resp.Fields[ef.name]
		if !ok {
			t.Errorf("Expected inherited field '%s' from BaseEntity in User", ef.name)
			continue
		}
		if field.Model == nil {
			t.Errorf("Inherited field '%s' has nil model", ef.name)
			continue
		}
		if field.Model.Kind != ef.kind {
			t.Errorf("Inherited field '%s': expected kind %s, got %s", ef.name, ef.kind, field.Model.Kind)
		}
		if field.Model.TypeName != ef.typeName {
			t.Errorf("Inherited field '%s': expected typeName %s, got %s", ef.name, ef.typeName, field.Model.TypeName)
		}
	}
}

func TestCollect_Swagger2_ApiParam(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Find Swagger2Controller endpoints
	var createEp, getByIdEp, searchEp *collector.ApiEndpoint
	for i := range endpoints {
		if endpoints[i].Folder != "Swagger2Controller" {
			continue
		}
		switch endpoints[i].Name {
		case "create":
			createEp = &endpoints[i]
		case "getById":
			getByIdEp = &endpoints[i]
		case "search":
			searchEp = &endpoints[i]
		}
	}

	// Test 1: @ApiOperation(httpMethod = "POST") with @RequestMapping
	if createEp == nil {
		t.Fatal("Expected create endpoint from Swagger2Controller")
	}
	if createEp.Method != "POST" {
		t.Errorf("Expected POST method from @ApiOperation(httpMethod), got %q", createEp.Method)
	}
	if createEp.Description != "Create item" {
		t.Errorf("Expected 'Create item' from @ApiOperation(value), got %q", createEp.Description)
	}

	// Test 2: @ApiParam description extraction
	params := make(map[string]collector.ApiParameter)
	for _, p := range createEp.Parameters {
		params[p.Name] = p
	}
	nameParam := params["name"]
	if nameParam.Description != "Item name" {
		t.Errorf("Expected 'Item name' from @ApiParam, got %q", nameParam.Description)
	}
	descParam := params["desc"]
	if descParam.Description != "Item description" {
		t.Errorf("Expected 'Item description' from @ApiParam(value), got %q", descParam.Description)
	}
	if descParam.Required {
		t.Error("Expected desc to be optional because @ApiParam(required=false)")
	}
	if descParam.Default != "N/A" {
		t.Errorf("Expected 'N/A' default from @ApiParam(defaultValue), got %q", descParam.Default)
	}

	// Test 3: @ApiParam with @PathVariable still resolves to path type
	if getByIdEp == nil {
		t.Fatal("Expected getById endpoint from Swagger2Controller")
	}
	pathParams := make(map[string]collector.ApiParameter)
	for _, p := range getByIdEp.Parameters {
		pathParams[p.Name] = p
	}
	idParam := pathParams["id"]
	if idParam.In != "path" {
		t.Errorf("Expected id to be 'path' type, got %q", idParam.In)
	}
	if idParam.Description != "Item ID" {
		t.Errorf("Expected 'Item ID' from @ApiParam, got %q", idParam.Description)
	}
	if getByIdEp.Description != "Get item by id\n\nReturns a single item" {
		t.Errorf("Expected combined @ApiOperation(value, notes), got %q", getByIdEp.Description)
	}

	// Test 4: @ApiParam without @RequestParam defaults to query
	if searchEp == nil {
		t.Fatal("Expected search endpoint from Swagger2Controller")
	}
	queryParams := make(map[string]collector.ApiParameter)
	for _, p := range searchEp.Parameters {
		queryParams[p.Name] = p
	}
	keywordParam := queryParams["keyword"]
	if keywordParam.In != "query" {
		t.Errorf("Expected keyword to default to 'query' via @ApiParam, got %q", keywordParam.In)
	}
	if keywordParam.Description != "Search keyword" {
		t.Errorf("Expected 'Search keyword' from @ApiParam, got %q", keywordParam.Description)
	}

	// Test 5: @RequestMapping without method or @ApiOperation defaults to GET
	if searchEp.Method != "GET" {
		t.Errorf("Expected GET as default for @RequestMapping without method, got %q", searchEp.Method)
	}
}

func TestCollect_Deduplication(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	// Collect all frameworks — ensures no duplicates from directory scanning
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: testdataDir,
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	seen := make(map[string]int)
	for _, ep := range endpoints {
		key := ep.Folder + "|" + ep.Path + "|" + ep.Method + "|" + ep.Name
		seen[key]++
	}
	for key, count := range seen {
		if count > 1 {
			t.Errorf("Duplicate endpoint: %s (count=%d)", key, count)
		}
	}
}

func TestSupportedLanguages(t *testing.T) {
	c := New()
	langs := c.SupportedLanguages()
	if len(langs) != 1 || langs[0] != "java" {
		t.Errorf("SupportedLanguages() = %v, want [java] — Kotlin is not parsed and must not be claimed", langs)
	}
}

func TestCollect_KotlinOnlyProject(t *testing.T) {
	c := New()

	// A Kotlin-only project must fail loudly instead of returning an empty
	// success (issue #142).
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "UserController.kt"), []byte("class UserController"), 0644); err != nil {
		t.Fatalf("Failed to write UserController.kt: %v", err)
	}

	endpoints, err := c.Collect(collector.CollectContext{SourceDir: dir})
	if err == nil {
		t.Fatal("Expected error for Kotlin-only project, got nil")
	}
	if len(endpoints) != 0 {
		t.Errorf("Expected no endpoints, got %d", len(endpoints))
	}
	if !strings.Contains(err.Error(), "Kotlin") {
		t.Errorf("Expected error to mention Kotlin, got %q", err.Error())
	}
}

func TestCollect_KotlinSourceFile(t *testing.T) {
	c := New()

	dir := t.TempDir()
	ktFile := filepath.Join(dir, "UserController.kt")
	if err := os.WriteFile(ktFile, []byte("class UserController"), 0644); err != nil {
		t.Fatalf("Failed to write UserController.kt: %v", err)
	}

	_, err := c.Collect(collector.CollectContext{SourceDir: dir, SourceFile: ktFile})
	if err == nil {
		t.Fatal("Expected error for explicit Kotlin source file, got nil")
	}
	if !strings.Contains(err.Error(), "Kotlin") {
		t.Errorf("Expected error to mention Kotlin, got %q", err.Error())
	}
}

func TestCollect_TagsAndHeaders(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: testdataDir,
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	byName := make(map[string]collector.ApiEndpoint, len(endpoints))
	for _, ep := range endpoints {
		byName[ep.Name] = ep
	}

	// Acceptance: @RequestHeader("Authorization") exports the header in
	// Headers under its wire name, and the endpoint is tagged with its
	// controller.
	currentUser, ok := byName["currentUser"]
	if !ok {
		t.Fatal("Expected currentUser endpoint from HeaderController")
	}
	if len(currentUser.Tags) != 1 || currentUser.Tags[0] != "HeaderController" {
		t.Errorf("currentUser Tags = %v, want [HeaderController]", currentUser.Tags)
	}
	if len(currentUser.Headers) != 1 {
		t.Fatalf("currentUser Headers = %+v, want exactly the Authorization header", currentUser.Headers)
	}
	auth := currentUser.Headers[0]
	if auth.Name != "Authorization" {
		t.Errorf("Authorization header name = %q, want %q", auth.Name, "Authorization")
	}
	if !auth.Required {
		t.Errorf("Authorization header should be required by default")
	}
	for _, p := range currentUser.Parameters {
		if p.In == "header" {
			t.Errorf("header parameter %q should live in Headers, not Parameters", p.Name)
		}
	}

	// An optional header with an explicit value= name and no default.
	version, ok := byName["version"]
	if !ok {
		t.Fatal("Expected version endpoint from HeaderController")
	}
	if len(version.Headers) != 1 || version.Headers[0].Name != "X-Api-Version" {
		t.Errorf("version Headers = %+v, want [X-Api-Version]", version.Headers)
	}
	if version.Headers[0].Required {
		t.Errorf("X-Api-Version header should be optional (required = false)")
	}

	// JAX-RS @HeaderParam lands in Headers too.
	requestId, ok := byName["requestId"]
	if !ok {
		t.Fatal("Expected requestId endpoint from HeaderResource")
	}
	if len(requestId.Tags) != 1 || requestId.Tags[0] != "HeaderResource" {
		t.Errorf("requestId Tags = %v, want [HeaderResource]", requestId.Tags)
	}
	if len(requestId.Headers) != 1 || requestId.Headers[0].Name != "X-Request-Id" {
		t.Errorf("requestId Headers = %+v, want [X-Request-Id]", requestId.Headers)
	}

	// Feign @RequestHeader lands in Headers too.
	session, ok := byName["session"]
	if !ok {
		t.Fatal("Expected session endpoint from HeaderUserClient")
	}
	if len(session.Tags) != 1 || session.Tags[0] != "HeaderUserClient" {
		t.Errorf("session Tags = %v, want [HeaderUserClient]", session.Tags)
	}
	if len(session.Headers) != 1 || session.Headers[0].Name != "X-Session-Token" {
		t.Errorf("session Headers = %+v, want [X-Session-Token]", session.Headers)
	}
}

// TestCollect_SchemaResolution_ReactiveController covers reactor publishers:
// the wrapper is plumbing, so the exported schema is the type argument.
// `Mono<T>` is `T` and `Flux<T>` is an array of `T`, the same way
// `ResponseEntity<T>` is unwrapped.
func TestCollect_SchemaResolution_ReactiveController(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	byName := make(map[string]collector.ApiEndpoint, len(endpoints))
	for _, ep := range endpoints {
		if ep.Folder == "ReactiveUserController" {
			byName[ep.Name] = ep
		}
	}

	responseBody := func(name string) *model.ObjectModel {
		t.Helper()
		ep, ok := byName[name]
		if !ok {
			t.Fatalf("Expected %s endpoint from ReactiveUserController", name)
		}
		if ep.Response == nil || ep.Response.Body == nil {
			t.Fatalf("%s has no response body", name)
		}
		return ep.Response.Body
	}

	// Mono<User> unwraps to User, own and inherited fields included.
	body := responseBody("getUser")
	if body.Kind != model.KindObject || body.TypeName != "User" {
		t.Fatalf("getUser response = %s/%s, want object/User", body.Kind, body.TypeName)
	}
	for _, name := range []string{"name", "email", "active", "id"} {
		if _, ok := body.Fields[name]; !ok {
			t.Errorf("getUser response is missing field %q", name)
		}
	}

	// Mono<Result<User>>: the wrapper is gone, Result carries the payload.
	body = responseBody("getProfile")
	if body.Kind != model.KindObject || body.TypeName != "Result" {
		t.Fatalf("getProfile response = %s/%s, want object/Result", body.Kind, body.TypeName)
	}
	if len(body.Fields) != 3 {
		t.Errorf("getProfile Result has %d fields, want code/message/data", len(body.Fields))
	}
	if code := body.Fields["code"]; code == nil || code.Model.TypeName != model.JsonTypeInt {
		t.Errorf("getProfile Result.code = %+v, want int", code)
	}
	data := body.Fields["data"]
	if data == nil || data.Model == nil {
		t.Fatal("getProfile Result.data is missing")
	}
	if data.Model.Kind != model.KindObject || data.Model.TypeName != "User" {
		t.Errorf("getProfile Result.data = %s/%s, want object/User", data.Model.Kind, data.Model.TypeName)
	}
	if _, ok := data.Model.Fields["email"]; !ok {
		t.Error("getProfile Result.data does not expand the User fields")
	}

	// Mono<Result<PageResult<User>>>: two levels of generics must resolve.
	body = responseBody("listUsers")
	data = body.Fields["data"]
	if data == nil || data.Model == nil {
		t.Fatal("listUsers Result.data is missing")
	}
	if data.Model.Kind != model.KindObject || data.Model.TypeName != "PageResult" {
		t.Fatalf("listUsers Result.data = %s/%s, want object/PageResult", data.Model.Kind, data.Model.TypeName)
	}
	for _, name := range []string{"total", "page", "size", "items"} {
		if _, ok := data.Model.Fields[name]; !ok {
			t.Errorf("listUsers PageResult is missing field %q", name)
		}
	}
	items := data.Model.Fields["items"]
	if items == nil || items.Model == nil || items.Model.Kind != model.KindArray {
		t.Fatalf("listUsers PageResult.items is not an array: %+v", items)
	}
	if items.Model.Items == nil || items.Model.Items.TypeName != "User" {
		t.Errorf("listUsers PageResult.items element = %+v, want User", items.Model.Items)
	}
	if _, ok := items.Model.Items.Fields["name"]; !ok {
		t.Error("listUsers PageResult.items does not expand the User fields")
	}

	// Mono<Result<Void>>: nothing is returned, so data is null — not an
	// unresolved `Void` wrapper.
	body = responseBody("deleteUser")
	data = body.Fields["data"]
	if data == nil || data.Model == nil {
		t.Fatal("deleteUser Result.data is missing")
	}
	if data.Model.Kind != model.KindSingle || data.Model.TypeName != model.JsonTypeNull {
		t.Errorf("deleteUser Result.data = %s/%s, want single/null", data.Model.Kind, data.Model.TypeName)
	}

	// Flux<User> is a JSON array, so the element fields are exported.
	body = responseBody("streamUsers")
	if body.Kind != model.KindArray {
		t.Fatalf("streamUsers response = %s, want array", body.Kind)
	}
	if body.Items == nil || body.Items.Kind != model.KindObject || body.Items.TypeName != "User" {
		t.Fatalf("streamUsers response element = %+v, want object/User", body.Items)
	}
	if _, ok := body.Items.Fields["email"]; !ok {
		t.Error("streamUsers response element does not expand the User fields")
	}
}

// TestCollect_JAXRSResponseFromBody covers the shape that used to lose the
// whole response section: the return type is Response and the payload only
// exists in the method body (issue #164).
func TestCollect_JAXRSResponseFromBody(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"jaxrs"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	byFolderAndName := make(map[string]*collector.ApiEndpoint)
	for i := range endpoints {
		byFolderAndName[endpoints[i].Folder+"."+endpoints[i].Name] = &endpoints[i]
	}

	// Response.ok(Result.success(user)) expands to the result wrapper with the
	// User DTO under data.
	for _, key := range []string{"ResponseResource.get", "ResponseResource.create"} {
		ep, ok := byFolderAndName[key]
		if !ok {
			t.Fatalf("Expected endpoint %q", key)
		}
		if ep.Response == nil {
			t.Fatalf("%s: Response is nil, the section would be missing from the export", key)
		}
		if ep.Response.Body == nil || !ep.Response.Body.IsObject() {
			t.Fatalf("%s: expected an object body, got %+v", key, ep.Response.Body)
		}
		data := ep.Response.Body.Fields["data"]
		if data == nil || data.Model == nil || !data.Model.IsObject() {
			t.Fatalf("%s: expected data to hold User, got %+v", key, data)
		}
		if _, ok := data.Model.Fields["email"]; !ok {
			t.Errorf("%s: expected User field 'email' under data", key)
		}
	}

	// A paged payload keeps PageResult and its item list.
	paged, ok := byFolderAndName["ResponseResource.paged"]
	if !ok {
		t.Fatal("Expected endpoint ResponseResource.paged")
	}
	if paged.Response == nil || paged.Response.Body == nil {
		t.Fatal("paged: Response body is nil")
	}
	page := paged.Response.Body.Fields["data"]
	if page == nil || page.Model == nil || !page.Model.IsObject() {
		t.Fatalf("paged: expected data to hold PageResult, got %+v", page)
	}
	if _, ok := page.Model.Fields["items"]; !ok {
		t.Error("paged: expected PageResult field 'items' under data")
	}

	// A body-built response with no entity still exports a section.
	del, ok := byFolderAndName["ResponseResource.delete"]
	if !ok {
		t.Fatal("Expected endpoint ResponseResource.delete")
	}
	if del.Response == nil {
		t.Error("delete: a Response with no entity must still export a response section")
	}
}

// TestCollect_RequestBodyName covers the bodies that carry no fields of their
// own: a scalar and a raw collection. A DTO names itself with its fields, but
// `@RequestBody String callbackData` has nothing to name it with except the
// parameter, and losing that name left the exported row either unnamed or
// named after an element index (issue #166).
func TestCollect_RequestBodyName(t *testing.T) {
	dir := t.TempDir()

	writeJava(t, dir, "ScalarController.java", `package com.example;

import org.springframework.web.bind.annotation.DeleteMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import java.util.List;

@RestController
@RequestMapping("/api/spring")
public class ScalarController {

    @PostMapping("/callback")
    public String paymentCallback(@RequestBody String callbackData) {
        return callbackData;
    }

    @DeleteMapping("/batch")
    public String batchDelete(@RequestBody List<Long> ids) {
        return "";
    }
}
`)

	writeJava(t, dir, "ScalarResource.java", `package com.example;

import javax.ws.rs.DELETE;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import java.util.List;

@Path("/api/jaxrs")
public class ScalarResource {

    @POST
    @Path("/callback")
    public String paymentCallback(String callbackData) {
        return callbackData;
    }

    @DELETE
    @Path("/batch")
    public String batchDelete(List<Long> ids) {
        return "";
    }
}
`)

	writeJava(t, dir, "ScalarClient.java", `package com.example;

import feign.RequestLine;
import java.util.List;

public interface ScalarClient {

    @RequestLine("POST /api/feign/callback")
    String paymentCallback(String callbackData);

    @RequestLine("DELETE /api/feign/batch")
    String batchDelete(List<Long> ids);
}
`)

	c := New()
	endpoints, err := c.Collect(collector.CollectContext{SourceDir: dir})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	byPath := make(map[string]*collector.ApiEndpoint, len(endpoints))
	for i := range endpoints {
		byPath[endpoints[i].Path] = &endpoints[i]
	}

	// name, then the shape the schema has to keep.
	tests := []struct {
		path string
		name string
		kind model.ObjectModelKind
	}{
		{"/api/spring/callback", "callbackData", model.KindSingle},
		{"/api/spring/batch", "ids", model.KindArray},
		{"/api/jaxrs/callback", "callbackData", model.KindSingle},
		{"/api/jaxrs/batch", "ids", model.KindArray},
		{"/api/feign/callback", "callbackData", model.KindSingle},
		{"/api/feign/batch", "ids", model.KindArray},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			ep, ok := byPath[tt.path]
			if !ok {
				t.Fatalf("Expected endpoint %q, got %d endpoints", tt.path, len(byPath))
			}
			if ep.RequestBody == nil {
				t.Fatalf("%s: no request body was exported at all", tt.path)
			}
			if ep.RequestBody.Name != tt.name {
				t.Errorf("%s: request body name = %q, want %q", tt.path, ep.RequestBody.Name, tt.name)
			}
			if ep.RequestBody.Body == nil {
				t.Fatalf("%s: request body has no schema", tt.path)
			}
			if ep.RequestBody.Body.Kind != tt.kind {
				t.Errorf("%s: request body kind = %s, want %s", tt.path, ep.RequestBody.Body.Kind, tt.kind)
			}
		})
	}
}

func writeJava(t *testing.T, dir, name, source string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0644); err != nil {
		t.Fatalf("Failed to write %s: %v", name, err)
	}
}

// TestCollect_ReportsOpaqueResponses covers the collector side of the
// diagnostic: an opaque payload is reported instead of disappearing.
func TestCollect_ReportsOpaqueResponses(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	if _, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"jaxrs"},
	}); err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	reporter, ok := c.(collector.WarningsReporter)
	if !ok {
		t.Fatal("JavaCollector must report the gaps it works around")
	}

	var found bool
	for _, warning := range reporter.Warnings() {
		if strings.Contains(warning, "ResponseResource.update") {
			found = true
		}
		if strings.Contains(warning, "ResponseResource.delete") {
			t.Errorf("an entity-less response is not a lost payload, got %q", warning)
		}
	}
	if !found {
		t.Errorf("Expected a warning for ResponseResource.update, got %v", reporter.Warnings())
	}
}

// TestCollect_WildcardReturnType covers #167 end to end: a `Result<?>` handler
// exports the payload the body names, and an open argument never reaches the
// exported model as the type name "?".
func TestCollect_WildcardReturnType(t *testing.T) {
	dir := t.TempDir()

	writeJava(t, dir, "Result.java", `package com.example.api;

public class Result<T> {
    private int code;
    private String message;
    private T data;

    public static <T> Result<T> success(T data) {
        Result<T> result = new Result<>();
        result.data = data;
        return result;
    }
}
`)

	writeJava(t, dir, "PageResult.java", `package com.example.api;

import java.util.List;

public class PageResult {
    private long total;
    private int page;
    private int size;
    private int pages;
    private List<String> records;
}
`)

	writeJava(t, dir, "OrderPageController.java", `package com.example.api;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/orders")
public class OrderPageController {

    @GetMapping("/page")
    public Result<?> page() {
        return Result.success(new PageResult());
    }

    @GetMapping("/open")
    public Result<?> open() {
        return Result.success(service.pageOrders(new PageRequest(1, 10)));
    }
}
`)

	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  dir,
		Frameworks: []string{"springmvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("Expected 2 endpoints, got %d", len(endpoints))
	}

	// The body builds `new PageResult()`, so the open argument is PageResult.
	paged := findEndpoint(t, endpoints, "page")
	if paged.Response == nil || paged.Response.Body == nil || !paged.Response.Body.IsObject() {
		t.Fatalf("Expected an object response, got %+v", paged.Response)
	}
	data := paged.Response.Body.Fields["data"]
	if data == nil || data.Model == nil || !data.Model.IsObject() {
		t.Fatalf("Expected data to expand PageResult, got %+v", data)
	}
	if data.Model.TypeName != "PageResult" {
		t.Errorf("Expected data typeName 'PageResult', got %q", data.Model.TypeName)
	}
	for _, fieldName := range []string{"records", "total", "page", "size", "pages"} {
		if data.Model.Fields[fieldName] == nil {
			t.Errorf("Expected PageResult field %q under data", fieldName)
		}
	}

	// The body names no payload, so the argument stays open: an empty object,
	// not the literal "?" and not the empty string a "?" demo produced.
	open := findEndpoint(t, endpoints, "open")
	if open.Response == nil || open.Response.Body == nil || !open.Response.Body.IsObject() {
		t.Fatalf("Expected an object response, got %+v", open.Response)
	}
	openData := open.Response.Body.Fields["data"]
	if openData == nil || openData.Model == nil || !openData.Model.IsObject() {
		t.Fatalf("Expected data to be an open object, got %+v", openData)
	}
	if openData.Model.TypeName != "object" {
		t.Errorf("Expected data typeName 'object', got %q", openData.Model.TypeName)
	}

	// Nothing anywhere in the exported schemas may present a wildcard as a
	// type name; it is an argument, and open payloads are objects.
	for _, endpoint := range endpoints {
		if endpoint.RequestBody != nil {
			assertNoWildcardTypeNames(t, endpoint.RequestBody.Body)
		}
		if endpoint.Response != nil {
			assertNoWildcardTypeNames(t, endpoint.Response.Body)
		}
	}
}

// assertNoWildcardTypeNames walks a model and fails on any type name that
// still carries a wildcard.
func assertNoWildcardTypeNames(t *testing.T, m *model.ObjectModel) {
	t.Helper()
	if m == nil {
		return
	}
	if strings.Contains(m.TypeName, "?") {
		t.Errorf("Exported type name %q still carries a wildcard", m.TypeName)
	}
	if m.KeyModel != nil {
		assertNoWildcardTypeNames(t, m.KeyModel)
	}
	if m.ValueModel != nil {
		assertNoWildcardTypeNames(t, m.ValueModel)
	}
	assertNoWildcardTypeNames(t, m.Items)
	for _, field := range m.Fields {
		if field == nil {
			continue
		}
		assertNoWildcardTypeNames(t, field.Model)
	}
}

// TestCollect_GetterDerivedResponseProperty covers #168 end to end: a response
// wrapper that derives part of its wire format from a getter exports that
// property on every endpoint, and the field-backed properties stay single.
func TestCollect_GetterDerivedResponseProperty(t *testing.T) {
	dir := t.TempDir()

	writeJava(t, dir, "Result.java", `package com.example.api;

public class Result<T> {
    private Integer code;
    private String message;
    private T data;

    public Integer getCode() { return code; }
    public String getMessage() { return message; }
    public T getData() { return data; }

    public boolean isSuccess() { return code != null && code == 200; }
}
`)

	writeJava(t, dir, "User.java", `package com.example.api;

public class User {
    private Long id;
    private String name;
}
`)

	writeJava(t, dir, "UserController.java", `package com.example.api;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/api/users")
public class UserController {

    @GetMapping("/{id}")
    public Result<User> get(@PathVariable Long id) {
        return null;
    }

    @GetMapping("/current")
    public Result<User> current() {
        return null;
    }
}
`)

	c := New()
	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  dir,
		Frameworks: []string{"springmvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if len(endpoints) != 2 {
		t.Fatalf("Expected 2 endpoints, got %d", len(endpoints))
	}

	for _, ep := range endpoints {
		if ep.Response == nil || ep.Response.Body == nil || !ep.Response.Body.IsObject() {
			t.Fatalf("%s: expected an object response, got %+v", ep.Name, ep.Response)
		}
		fields := ep.Response.Body.Fields
		if len(fields) != 4 {
			t.Errorf("%s: expected code/message/data/success, got %d fields", ep.Name, len(fields))
		}
		success := fields["success"]
		if success == nil || success.Model == nil {
			t.Errorf("%s: expected a 'success' property derived from isSuccess()", ep.Name)
			continue
		}
		if success.Model.TypeName != model.JsonTypeBoolean {
			t.Errorf("%s: success type = %q, want boolean", ep.Name, success.Model.TypeName)
		}
		if success.Demo != "true" {
			t.Errorf("%s: success demo = %q, want the enabling case", ep.Name, success.Demo)
		}
		for _, name := range []string{"code", "message", "data"} {
			if fields[name] == nil {
				t.Errorf("%s: expected declared property %q", ep.Name, name)
			}
		}
	}
}

// TestCollect_ParameterTypeName covers issue #169's core gap: the declared
// Java type travels to the canonical model, so a caller can tell a Long id
// from a String id.
func TestCollect_ParameterTypeName(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir:  testdataDir,
		Frameworks: []string{"spring-mvc"},
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	getUser := findEndpoint(t, endpoints, "getUser")
	id := findParameter(t, getUser, "id")
	if id.TypeName != "Long" {
		t.Errorf("getUser id typeName = %q, want Long", id.TypeName)
	}
	if id.Type != "text" {
		t.Errorf("getUser id type = %q, want text", id.Type)
	}

	listUsers := findEndpoint(t, endpoints, "listUsers")
	page := findParameter(t, listUsers, "page")
	if page.TypeName != "int" {
		t.Errorf("listUsers page typeName = %q, want int", page.TypeName)
	}
}

// TestCollect_MultipartFileIsFormParam covers the correctness half of issue
// #169: a MultipartFile argument rides in a multipart form body, so it is a
// form parameter with kind file — never a query parameter, where a reader
// following the document would send ?file=.
func TestCollect_MultipartFileIsFormParam(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: testdataDir,
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	upload := findEndpoint(t, endpoints, "upload")
	if len(upload.Parameters) != 2 {
		t.Fatalf("upload parameters = %+v, want the file part and the description", upload.Parameters)
	}
	file := findParameter(t, upload, "file")
	if file.In != "form" {
		t.Errorf("upload file in = %q, want form", file.In)
	}
	if file.Type != "file" {
		t.Errorf("upload file type = %q, want file", file.Type)
	}
	if file.TypeName != "MultipartFile" {
		t.Errorf("upload file typeName = %q, want MultipartFile", file.TypeName)
	}
	if !file.Required {
		t.Error("upload file should stay required: @RequestParam defaults to required")
	}
	// A plain @RequestParam next to the file is not a file part, and Spring
	// binds it from the query string as readily as from the form, so it stays
	// where the binding annotation puts it.
	description := findParameter(t, upload, "description")
	if description.In != "query" || description.Type != "text" {
		t.Errorf("upload description = %+v, want a text query parameter", description)
	}

	// @RequestPart is the other multipart binding; it is a form part too.
	uploadParts := findEndpoint(t, endpoints, "uploadParts")
	if len(uploadParts.Parameters) != 2 {
		t.Fatalf("uploadParts parameters = %+v, want the avatar part and the bio part", uploadParts.Parameters)
	}
	avatar := findParameter(t, uploadParts, "avatar")
	if avatar.In != "form" || avatar.Type != "file" {
		t.Errorf("uploadParts avatar = %+v, want a form file parameter", avatar)
	}
	bio := findParameter(t, uploadParts, "bio")
	if bio.In != "form" || bio.Type != "text" {
		t.Errorf("uploadParts bio = %+v, want a form text parameter", bio)
	}
}

// TestCollect_DeclaredMediaTypes covers the media type half of issue #169: the
// media types the source declares reach the model as wire media types — the
// MediaType constant reference is resolved, not exported verbatim — and an
// endpoint that declares none stays empty.
func TestCollect_DeclaredMediaTypes(t *testing.T) {
	c := New()
	testdataDir, _ := filepath.Abs("testdata")

	endpoints, err := c.Collect(collector.CollectContext{
		SourceDir: testdataDir,
	})
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}

	// Spring: the mapping annotation declares both directions.
	upload := findEndpoint(t, endpoints, "upload")
	if len(upload.Consumes) != 1 || upload.Consumes[0] != "multipart/form-data" {
		t.Errorf("upload consumes = %v, want [multipart/form-data]", upload.Consumes)
	}
	if len(upload.Produces) != 1 || upload.Produces[0] != "application/json" {
		t.Errorf("upload produces = %v, want [application/json]", upload.Produces)
	}

	// Spring: a literal value is taken as written.
	uploadParts := findEndpoint(t, endpoints, "uploadParts")
	if len(uploadParts.Consumes) != 1 || uploadParts.Consumes[0] != "multipart/form-data" {
		t.Errorf("uploadParts consumes = %v, want [multipart/form-data]", uploadParts.Consumes)
	}
	if len(uploadParts.Produces) != 0 {
		t.Errorf("uploadParts produces = %v, want none", uploadParts.Produces)
	}

	// JAX-RS: the class-level declaration is inherited by every method, and a
	// method declaration overrides it.
	byFolder := make(map[string]collector.ApiEndpoint)
	for _, ep := range endpoints {
		if ep.Folder == "UploadResource" {
			byFolder[ep.Name] = ep
		}
	}
	form, ok := byFolder["upload"]
	if !ok {
		t.Fatal("Expected the JAX-RS upload endpoint")
	}
	if len(form.Consumes) != 1 || form.Consumes[0] != "multipart/form-data" {
		t.Errorf("jaxrs upload consumes = %v, want [multipart/form-data]", form.Consumes)
	}
	if len(form.Produces) != 1 || form.Produces[0] != "application/json" {
		t.Errorf("jaxrs upload produces = %v, want [application/json]", form.Produces)
	}
	if file := findParameter(t, &form, "file"); file.In != "form" || file.Type != "file" {
		t.Errorf("jaxrs upload file = %+v, want a form file parameter", file)
	}

	plain, ok := byFolder["uploadPlain"]
	if !ok {
		t.Fatal("Expected the JAX-RS uploadPlain endpoint")
	}
	if len(plain.Consumes) != 1 || plain.Consumes[0] != "application/x-www-form-urlencoded" {
		t.Errorf("uploadPlain consumes = %v, want [application/x-www-form-urlencoded]", plain.Consumes)
	}
	if len(plain.Produces) != 1 || plain.Produces[0] != "text/plain" {
		t.Errorf("uploadPlain produces = %v, want [text/plain]", plain.Produces)
	}
	if plain.Response == nil || plain.Response.MediaType != "text/plain" {
		t.Errorf("uploadPlain response media type = %v, want text/plain", plain.Response)
	}

	// A handler whose mapping declares nothing gets nothing: no media type is
	// invented on the caller's behalf.
	getUser := findEndpoint(t, endpoints, "getUser")
	if len(getUser.Consumes) != 0 || len(getUser.Produces) != 0 {
		t.Errorf("getUser media types = %v / %v, want none declared", getUser.Consumes, getUser.Produces)
	}
}

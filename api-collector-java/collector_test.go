package javacollector

import (
	"path/filepath"
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

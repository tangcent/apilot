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

	attrsField := resp.Fields["attributes"]
	if attrsField != nil && attrsField.Model != nil && attrsField.Model.ValueModel != nil {
		if !attrsField.Model.ValueModel.IsSingle() {
			t.Errorf("Expected attributes value model to be single, got kind=%s", attrsField.Model.ValueModel.Kind)
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
		{"createdAt", model.KindSingle, "LocalDateTime"},
		{"updatedAt", model.KindSingle, "LocalDateTime"},
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

package postman

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tangcent/apilot/api-formatter-postman/model"
)

func TestCreateCollection_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/collections" {
			t.Errorf("Expected /collections, got %s", r.URL.Path)
		}
		if r.Header.Get("X-Api-Key") != "test-api-key" {
			t.Errorf("Expected X-Api-Key 'test-api-key', got %q", r.Header.Get("X-Api-Key"))
		}

		resp := CreateCollectionResponse{}
		resp.Collection.ID = "col-123"
		resp.Collection.Name = "Test Collection"
		resp.Collection.UID = "col-123-uid"
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := PostmanClient{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		HTTP:    server.Client(),
	}

	col := model.Collection{
		Info: model.Info{Name: "Test Collection", Schema: postmanSchema},
		Item: []model.ItemGroup{},
	}

	result, err := client.CreateCollection("", col)
	if err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}
	if result.Collection.ID != "col-123" {
		t.Errorf("Expected ID 'col-123', got %q", result.Collection.ID)
	}
	if result.Collection.UID != "col-123-uid" {
		t.Errorf("Expected UID 'col-123-uid', got %q", result.Collection.UID)
	}
}

func TestCreateCollection_WithWorkspace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("workspace") != "ws-123" {
			t.Errorf("Expected workspace 'ws-123', got %q", r.URL.Query().Get("workspace"))
		}
		resp := CreateCollectionResponse{}
		resp.Collection.ID = "col-456"
		resp.Collection.UID = "col-456-uid"
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := PostmanClient{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		HTTP:    server.Client(),
	}

	col := model.Collection{
		Info: model.Info{Name: "Test", Schema: postmanSchema},
		Item: []model.ItemGroup{},
	}

	result, err := client.CreateCollection("ws-123", col)
	if err != nil {
		t.Fatalf("CreateCollection() error = %v", err)
	}
	if result.Collection.ID != "col-456" {
		t.Errorf("Expected ID 'col-456', got %q", result.Collection.ID)
	}
}

func TestCreateCollection_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"name":    "AuthenticationError",
			"message": "Invalid API key",
		})
	}))
	defer server.Close()

	client := PostmanClient{
		APIKey:  "invalid-key",
		BaseURL: server.URL,
		HTTP:    server.Client(),
	}

	col := model.Collection{
		Info: model.Info{Name: "Test", Schema: postmanSchema},
		Item: []model.ItemGroup{},
	}

	_, err := client.CreateCollection("", col)
	if err == nil {
		t.Fatal("Expected error for API error response, got nil")
	}
	apiErr, ok := err.(*PostmanAPIError)
	if !ok {
		t.Fatalf("Expected *PostmanAPIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("Expected status 401, got %d", apiErr.StatusCode)
	}
}

func TestUpdateCollection_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/collections/col-uid-123" {
			t.Errorf("Expected /collections/col-uid-123, got %s", r.URL.Path)
		}

		resp := UpdateCollectionResponse{}
		resp.Collection.ID = "col-123"
		resp.Collection.Name = "Updated Collection"
		resp.Collection.UID = "col-uid-123"
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := PostmanClient{
		APIKey:  "test-api-key",
		BaseURL: server.URL,
		HTTP:    server.Client(),
	}

	col := model.Collection{
		Info: model.Info{Name: "Updated Collection", Schema: postmanSchema},
		Item: []model.ItemGroup{},
	}

	result, err := client.UpdateCollection("col-uid-123", col)
	if err != nil {
		t.Fatalf("UpdateCollection() error = %v", err)
	}
	if result.Collection.ID != "col-123" {
		t.Errorf("Expected ID 'col-123', got %q", result.Collection.ID)
	}
}

func TestNewPostmanClient(t *testing.T) {
	client := newPostmanClient("my-key")
	if client.APIKey != "my-key" {
		t.Errorf("Expected APIKey 'my-key', got %q", client.APIKey)
	}
	if client.BaseURL != defaultPostmanAPIBase {
		t.Errorf("Expected BaseURL %q, got %q", defaultPostmanAPIBase, client.BaseURL)
	}
	if client.HTTP == nil {
		t.Error("Expected HTTP client to be initialized")
	}
}

func TestPostmanAPIError_Error(t *testing.T) {
	err := &PostmanAPIError{
		StatusCode: 401,
		Name:       "AuthenticationError",
		Message:    "Invalid API key",
	}
	expected := "postman api error (status 401): AuthenticationError - Invalid API key"
	if err.Error() != expected {
		t.Errorf("Expected %q, got %q", expected, err.Error())
	}
}

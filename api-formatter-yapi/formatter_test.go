package yapi

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	formatter "github.com/tangcent/apilot/api-formatter"
	model "github.com/tangcent/apilot/api-model"
)

// fakeYapi answers the open-API calls the formatter makes for one project.
type fakeYapi struct {
	mu         sync.Mutex
	token      string
	categories map[string]int64
	existing   map[string]int64 // "METHOD path" -> id
	nextID     int64
	saved      []map[string]any
	failPath   string
}

func (f *fakeYapi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	reply := func(data any) {
		_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 0, "errmsg": "成功！", "data": data})
	}
	fail := func(msg string) {
		_ = json.NewEncoder(w).Encode(map[string]any{"errcode": 40011, "errmsg": msg})
	}

	token := r.URL.Query().Get("token")
	var body map[string]any
	if r.Method == http.MethodPost {
		data, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(data, &body)
		token, _ = body["token"].(string)
	}
	if token != f.token {
		fail("请登录...")
		return
	}

	switch r.URL.Path {
	case "/api/project/get":
		reply(map[string]any{"_id": 7})
	case "/api/interface/getCatMenu":
		var cats []map[string]any
		for name, id := range f.categories {
			cats = append(cats, map[string]any{"_id": id, "name": name})
		}
		reply(cats)
	case "/api/interface/add_cat":
		f.nextID++
		f.categories[body["name"].(string)] = f.nextID
		reply(map[string]any{"_id": f.nextID})
	case "/api/interface/list":
		var list []map[string]any
		for key, id := range f.existing {
			parts := strings.SplitN(key, " ", 2)
			list = append(list, map[string]any{"_id": id, "method": parts[0], "path": parts[1]})
		}
		reply(map[string]any{"count": len(list), "list": list})
	case "/api/interface/save":
		if body["path"] == f.failPath {
			fail("path conflict")
			return
		}
		f.saved = append(f.saved, body)
		reply([]any{})
	default:
		http.NotFound(w, r)
	}
}

type mapSettings map[string]string

func (m mapSettings) Get(key string) string { return m[key] }

func TestFormatUploads(t *testing.T) {
	yapi := &fakeYapi{
		token:      "secret",
		categories: map[string]int64{"Order management.": 11},
		existing:   map[string]int64{"GET /orders": 501},
		nextID:     100,
	}
	server := httptest.NewServer(yapi)
	defer server.Close()

	list := model.ApiEndpoint{Name: "list", Folder: "OrderController", FolderDescription: "Order management.", Path: "/orders", Method: "GET"}
	ping := model.ApiEndpoint{Name: "ping", Folder: "HealthController", Path: "/ping", Method: "GET"}

	f := &YapiFormatter{client: server.Client()}
	out, err := f.Format([]model.ApiEndpoint{orderEndpoint(), list, ping}, formatter.FormatOptions{
		Settings: mapSettings{"yapi.server": server.URL + "/", "yapi.token": "secret"},
	})
	if err != nil {
		t.Fatalf("Format: %v", err)
	}

	var res Result
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("result is not JSON: %v\n%s", err, out)
	}
	if res.Project != 7 || res.Created != 2 || res.Updated != 1 || res.Failed != 0 {
		t.Fatalf("result = %+v", res)
	}
	if len(yapi.saved) != 3 {
		t.Fatalf("saved %d interfaces", len(yapi.saved))
	}
	byPath := map[string]map[string]any{}
	for _, s := range yapi.saved {
		byPath[s["path"].(string)] = s
	}
	// An existing interface is saved with its id; every save carries the token
	// and the category it belongs to, which is created when missing.
	if byPath["/orders"]["id"] != float64(501) || byPath["/orders"]["catid"] != float64(11) {
		t.Errorf("/orders saved as %v", byPath["/orders"])
	}
	if _, ok := byPath["/orders/{orderNo}"]["id"]; ok {
		t.Errorf("a new interface must not carry an id: %v", byPath["/orders/{orderNo}"])
	}
	if byPath["/ping"]["catid"] != float64(yapi.categories["HealthController"]) || yapi.categories["HealthController"] == 0 {
		t.Errorf("/ping category: saved %v, categories %v", byPath["/ping"]["catid"], yapi.categories)
	}
	for path, s := range byPath {
		if s["token"] != "secret" {
			t.Errorf("%s saved without the token", path)
		}
	}
}

func TestFormatReportsFailedSaves(t *testing.T) {
	yapi := &fakeYapi{token: "secret", categories: map[string]int64{}, existing: map[string]int64{}, failPath: "/ping"}
	server := httptest.NewServer(yapi)
	defer server.Close()

	f := &YapiFormatter{client: server.Client()}
	_, err := f.Format([]model.ApiEndpoint{
		{Name: "ping", Path: "/ping", Method: "GET"},
		{Name: "pong", Path: "/pong", Method: "GET"},
	}, formatter.FormatOptions{Params: json.RawMessage(`{"server":"` + server.URL + `","token":"secret"}`)})
	if err == nil || !strings.Contains(err.Error(), "1 of 2") || !strings.Contains(err.Error(), "path conflict") {
		t.Fatalf("expected a partial-failure error, got %v", err)
	}
}

func TestFormatRejectsBadToken(t *testing.T) {
	server := httptest.NewServer(&fakeYapi{token: "secret"})
	defer server.Close()
	f := &YapiFormatter{client: server.Client()}
	_, err := f.Format([]model.ApiEndpoint{{Name: "ping", Path: "/ping", Method: "GET"}},
		formatter.FormatOptions{Params: json.RawMessage(`{"server":"` + server.URL + `","token":"wrong"}`)})
	if err == nil || !strings.Contains(err.Error(), "/api/project/get") {
		t.Fatalf("expected the token check to fail, got %v", err)
	}
}

func TestFormatWithoutServerPrintsImportDocument(t *testing.T) {
	f := New()
	out, err := f.Format([]model.ApiEndpoint{
		orderEndpoint(),
		{Name: "list", Folder: "OrderController", FolderDescription: "Order management.", Path: "/orders", Method: "GET"},
		{Name: "ping", Folder: "HealthController", Path: "/ping", Method: "GET"},
	}, formatter.FormatOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var doc []struct {
		Name string           `json:"name"`
		List []map[string]any `json:"list"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("not an import document: %v\n%s", err, out)
	}
	if len(doc) != 2 || doc[0].Name != "Order management." || len(doc[0].List) != 2 || doc[1].Name != "HealthController" {
		t.Fatalf("import document = %+v", doc)
	}
	if _, ok := doc[0].List[0]["token"]; ok {
		t.Errorf("an import document carries no token: %v", doc[0].List[0])
	}

	empty, err := f.Format(nil, formatter.FormatOptions{})
	if err != nil || strings.TrimSpace(string(empty)) != "[]" {
		t.Fatalf("an empty export is an empty document, got %q, %v", empty, err)
	}
}

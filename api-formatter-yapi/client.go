package yapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// client calls YApi's open API with a project token.
type client struct {
	server string
	token  string
	http   *http.Client
}

type envelope struct {
	Errcode int             `json:"errcode"`
	Errmsg  string          `json:"errmsg"`
	Data    json.RawMessage `json:"data"`
}

func (c *client) decode(path string, resp *http.Response, out any) error {
	defer resp.Body.Close()
	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("%s: HTTP %d, unreadable answer: %w", path, resp.StatusCode, err)
	}
	if env.Errcode != 0 {
		return fmt.Errorf("%s: %s (errcode %d)", path, env.Errmsg, env.Errcode)
	}
	if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, out); err != nil {
			return fmt.Errorf("%s: unexpected data: %w", path, err)
		}
	}
	return nil
}

func (c *client) get(path string, query url.Values, out any) error {
	query.Set("token", c.token)
	resp, err := c.http.Get(c.server + path + "?" + query.Encode())
	if err != nil {
		return fmt.Errorf("%s: %w", path, redactHTTPError(err))
	}
	return c.decode(path, resp, out)
}

// redactHTTPError keeps an HTTP error's underlying cause but drops the
// *url.Error wrapper, whose message embeds the full request URL. A GET query
// here carries the project token, which must never reach the logs.
func redactHTTPError(err error) error {
	if ue, ok := err.(*url.Error); ok {
		return ue.Err
	}
	return err
}

func (c *client) post(path string, body any, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	resp, err := c.http.Post(c.server+path, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return c.decode(path, resp, out)
}

// projectID returns the project the token belongs to.
func (c *client) projectID() (int64, error) {
	var project struct {
		ID int64 `json:"_id"`
	}
	if err := c.get("/api/project/get", url.Values{}, &project); err != nil {
		return 0, err
	}
	if project.ID == 0 {
		return 0, fmt.Errorf("/api/project/get: the token does not name a project")
	}
	return project.ID, nil
}

// categories returns the project's categories by name.
func (c *client) categories(projectID int64) (map[string]int64, error) {
	var cats []struct {
		ID   int64  `json:"_id"`
		Name string `json:"name"`
	}
	q := url.Values{"project_id": {strconv.FormatInt(projectID, 10)}}
	if err := c.get("/api/interface/getCatMenu", q, &cats); err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(cats))
	for _, cat := range cats {
		if _, ok := out[cat.Name]; !ok {
			out[cat.Name] = cat.ID
		}
	}
	return out, nil
}

func (c *client) addCategory(projectID int64, name string) (int64, error) {
	var cat struct {
		ID int64 `json:"_id"`
	}
	body := map[string]any{"desc": "", "project_id": projectID, "name": name, "token": c.token}
	if err := c.post("/api/interface/add_cat", body, &cat); err != nil {
		return 0, err
	}
	return cat.ID, nil
}

// existing returns the project's interfaces, keyed by "METHOD path".
func (c *client) existing(projectID int64) (map[string]int64, error) {
	out := map[string]int64{}
	const limit = 1000
	for page := 1; ; page++ {
		var data struct {
			Count int `json:"count"`
			List  []struct {
				ID     int64  `json:"_id"`
				Path   string `json:"path"`
				Method string `json:"method"`
			} `json:"list"`
		}
		q := url.Values{
			"project_id": {strconv.FormatInt(projectID, 10)},
			"page":       {strconv.Itoa(page)},
			"limit":      {strconv.Itoa(limit)},
		}
		if err := c.get("/api/interface/list", q, &data); err != nil {
			return nil, err
		}
		for _, it := range data.List {
			out[strings.ToUpper(it.Method)+" "+it.Path] = it.ID
		}
		if len(data.List) < limit || len(out) >= data.Count {
			return out, nil
		}
	}
}

func (c *client) save(it Interface) error {
	return c.post("/api/interface/save", it, nil)
}

// Result reports what an upload did.
type Result struct {
	Server   string       `json:"server"`
	Project  int64        `json:"project"`
	Created  int          `json:"created"`
	Updated  int          `json:"updated"`
	Failed   int          `json:"failed"`
	Rules    []string     `json:"rules,omitempty"`
	Warnings []string     `json:"warnings,omitempty"`
	Items    []ResultItem `json:"items"`
}

// ResultItem reports one interface: created, updated or failed.
type ResultItem struct {
	Status   string `json:"status"`
	Method   string `json:"method"`
	Path     string `json:"path"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Reason   string `json:"reason,omitempty"`
}

// upload saves the interfaces to the token's project. An interface whose path
// and method already exist in the project is updated and moved into its
// category, as YApi's save does; any other is created. Missing categories are
// created by name.
func upload(c *client, interfaces []Interface) (*Result, error) {
	projectID, err := c.projectID()
	if err != nil {
		return nil, err
	}
	cats, err := c.categories(projectID)
	if err != nil {
		return nil, err
	}
	existing, err := c.existing(projectID)
	if err != nil {
		return nil, err
	}

	res := &Result{Server: c.server, Project: projectID, Items: []ResultItem{}}
	for _, it := range interfaces {
		item := ResultItem{Method: strings.ToUpper(it.Method), Path: it.Path, Title: it.Title, Category: it.category}
		catID, ok := cats[it.category]
		if !ok {
			catID, err = c.addCategory(projectID, it.category)
			if err != nil {
				item.Status, item.Reason = "failed", err.Error()
				res.Failed++
				res.Items = append(res.Items, item)
				continue
			}
			cats[it.category] = catID
		}
		it.Token, it.CatID = c.token, catID
		item.Status = "created"
		if id, ok := existing[item.Method+" "+it.Path]; ok {
			it.ID = id
			item.Status = "updated"
		}
		if err := c.save(it); err != nil {
			item.Status, item.Reason = "failed", err.Error()
			res.Failed++
		} else if item.Status == "updated" {
			res.Updated++
		} else {
			res.Created++
		}
		res.Items = append(res.Items, item)
	}
	return res, nil
}

// Package model contains typed Go structs for the Postman Collection v2.1 format.
package model

import "encoding/json"

// Collection is the top-level Postman Collection v2.1 document.
type Collection struct {
	Info     Info       `json:"info"`
	Item     []Item     `json:"item"`
	Event    []Event    `json:"event,omitempty"`
	Variable []Variable `json:"variable,omitempty"`
}

// Info holds collection metadata.
type Info struct {
	Name        string `json:"name"`
	Schema      string `json:"schema"`
	Description string `json:"description,omitempty"`
}

// Item represents either a folder (has Item sub-slice) or a single API request (has Request).
// Postman distinguishes them by the presence of "request" vs "item" in JSON.
// Custom MarshalJSON ensures correct serialization.
type Item struct {
	Name        string     `json:"name"`
	Item        []Item     `json:"item,omitempty"`
	Request     *Request   `json:"request,omitempty"`
	Response    []Response `json:"response,omitempty"`
	Event       []Event    `json:"event,omitempty"`
	Description string     `json:"description,omitempty"`
}

// IsFolder returns true if this item is a folder (has sub-items, no request).
func (i Item) IsFolder() bool {
	return i.Request == nil
}

// MarshalJSON implements custom JSON serialization for Item.
// Folders: output name, item, description, event — NOT request.
// API items: output name, request, response, event — NOT item.
func (i Item) MarshalJSON() ([]byte, error) {
	type ItemAlias Item
	alias := ItemAlias(i)

	if i.IsFolder() {
		obj := make(map[string]any)
		obj["name"] = i.Name
		if i.Description != "" {
			obj["description"] = i.Description
		}
		if i.Item != nil {
			obj["item"] = i.Item
		}
		if len(i.Event) > 0 {
			obj["event"] = i.Event
		}
		return json.Marshal(obj)
	}

	obj := make(map[string]any)
	obj["name"] = i.Name
	obj["request"] = alias.Request
	if alias.Response != nil {
		obj["response"] = alias.Response
	} else {
		obj["response"] = []struct{}{}
	}
	if i.Description != "" {
		obj["description"] = i.Description
	}
	if len(i.Event) > 0 {
		obj["event"] = i.Event
	}
	return json.Marshal(obj)
}

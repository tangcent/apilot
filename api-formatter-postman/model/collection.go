// Package model contains typed Go structs for the Postman Collection v2.1 format.
package model

// Collection is the top-level Postman Collection v2.1 document.
type Collection struct {
	Info  Info        `json:"info"`
	Item  []ItemGroup `json:"item"`
}

// Info holds collection metadata.
type Info struct {
	Name   string `json:"name"`
	Schema string `json:"schema"` // always "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
}

// ItemGroup is a folder containing items or nested folders.
type ItemGroup struct {
	Name string `json:"name"`
	Item []Item `json:"item"`
}

// Item represents a single Postman request.
type Item struct {
	Name    string  `json:"name"`
	Request Request `json:"request"`
}

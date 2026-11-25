package parser

import (
	"reflect"
	"testing"
)

// A Kotlin import may end in a semicolon, as a Java one does, and may carry an
// alias. The semicolon is no part of the import: with it kept,
// `import com.example.lib.PageData;` named a type PageData; that no
// declaration has. The alias is the name the import binds, so it is kept: a
// renaming import binds the new name only.
func TestImportsAreReadWithoutTheirSemicolonAndKeepTheirAlias(t *testing.T) {
	result := parseSource(t, "src/Api.kt", `package com.example.app

import com.example.lib.PageData;
import com.example.lib.Wrapper
import com.example.lib.Result as LibResult;
import com.example.model.*;
import org.springframework.web.bind.annotation.*

class Api
`)
	want := []Import{
		{Path: "com.example.lib.PageData"},
		{Path: "com.example.lib.Wrapper"},
		{Path: "com.example.lib.Result", Alias: "LibResult"},
		{Path: "com.example.model", OnDemand: true},
		{Path: "org.springframework.web.bind.annotation", OnDemand: true},
	}
	if got := result.Classes[0].Imports; !reflect.DeepEqual(got, want) {
		t.Errorf("imports:\n got  %+v\n want %+v", got, want)
	}
}

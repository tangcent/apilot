package parser

import "testing"

func TestParameterTypes(t *testing.T) {
	p, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	classes, err := p.ParseSource([]byte(`package com.example;

public class Api {
    public void call(Long[] ids, java.time.LocalDate day, org.springframework.web.multipart.MultipartFile[] files, int n) {}
}
`))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"ids":   "Long[]",
		"day":   "java.time.LocalDate",
		"files": "org.springframework.web.multipart.MultipartFile[]",
		"n":     "int",
	}
	params := classes[0].Methods[0].Parameters
	if len(params) != len(want) {
		t.Fatalf("parameters = %+v", params)
	}
	for _, param := range params {
		if param.Type != want[param.Name] {
			t.Errorf("%s: type = %q, want %q", param.Name, param.Type, want[param.Name])
		}
	}
}

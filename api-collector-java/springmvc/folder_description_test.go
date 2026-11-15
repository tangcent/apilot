package springmvc

import (
	"testing"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

func TestParser_ControllerDescription(t *testing.T) {
	jp, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		t.Fatalf("Failed to create java parser: %v", err)
	}
	defer jp.Close()

	classes, err := jp.ParseSource([]byte(`package com.example.api;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

/**
 * Order management.
 *
 * @author someone
 */
@RestController
public class OrderController {
    @GetMapping("/orders")
    public String list() {
        return "";
    }
}
`))
	if err != nil {
		t.Fatalf("Failed to parse source: %v", err)
	}
	controllers := NewParser().ExtractControllers([]parser.ParseResult{{FilePath: "OrderController.java", Classes: classes}})
	if len(controllers) != 1 {
		t.Fatalf("expected 1 controller, got %d", len(controllers))
	}
	if got := controllers[0].Description; got != "Order management." {
		t.Fatalf("description = %q", got)
	}
}

package com.example.scope.a;

import com.fasterxml.jackson.annotation.JsonValue;

/** Jackson writes what the annotated method returns. */
public enum Code {
    X("x"),
    Y("y");

    private final String text;

    Code(String text) {
        this.text = text;
    }

    @JsonValue
    public String getText() {
        return text;
    }
}

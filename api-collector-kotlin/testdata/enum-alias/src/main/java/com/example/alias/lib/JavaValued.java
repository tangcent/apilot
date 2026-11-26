package com.example.alias.lib;

import com.fasterxml.jackson.annotation.JsonValue;

/** An interface written in Java that an enum written in Kotlin implements. */
public interface JavaValued {
    @JsonValue
    default String wire() {
        return "";
    }
}

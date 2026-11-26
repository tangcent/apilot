package com.example.alias.lib;

/** The same member as JavaValued, without @JsonValue. */
public interface JavaCoded {
    default String wire() {
        return "";
    }
}

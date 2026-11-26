package com.example.cross;

import com.fasterxml.jackson.annotation.JsonValue;

interface JavaCoded {
    @JsonValue
    String getCode();
}

enum JavaOn implements KtCoded {
    A,
    B;

    public String getCode() {
        return "x";
    }
}

enum JavaOff implements KtCoded {
    A,
    B;

    @JsonValue(false)
    public String getCode() {
        return "x";
    }
}

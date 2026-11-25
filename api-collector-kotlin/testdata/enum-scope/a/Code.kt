package com.example.scope.a

import com.fasterxml.jackson.annotation.JsonValue

/** Jackson writes what the annotated getter returns. */
enum class Code(@get:JsonValue val text: String) {
    X("x"),
    Y("y"),
}

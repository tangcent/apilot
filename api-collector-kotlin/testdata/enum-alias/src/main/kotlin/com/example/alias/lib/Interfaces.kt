package com.example.alias.lib

import com.fasterxml.jackson.annotation.JsonValue

/** Jackson writes what the member returns, for an enum that implements this. */
interface Valued {
    @get:JsonValue
    val wire: String get() = ""
}

/** The same member as Valued, without @JsonValue: an enum that implements this is written as the name of its entry. */
interface Coded {
    val wire: String get() = ""
}

interface TypedValued<T> {
    @get:JsonValue
    val wire: T? get() = null
}

interface TypedCoded<T> {
    val wire: T? get() = null
}

interface Base {
    @get:JsonValue
    val wire: String get() = ""
}

interface BaseCoded {
    val wire: String get() = ""
}

class Outer {
    interface Valued {
        @get:JsonValue
        val wire: String get() = ""
    }

    interface Coded {
        val wire: String get() = ""
    }
}

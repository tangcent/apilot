package com.example.alias.use

import com.fasterxml.jackson.annotation.JsonValue

/** The interface is declared in the package of the enum, and nothing imports it. */
interface SameValued {
    @get:JsonValue
    val wire: String get() = ""
}

interface SameCoded {
    val wire: String get() = ""
}

enum class SameJson : SameValued {
    A,
    B
}

enum class SamePlain : SameCoded {
    SAME_ONE,
    SAME_TWO
}

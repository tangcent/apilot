package com.example.alias.decoy

import com.fasterxml.jackson.annotation.JsonValue

/** Shares its simple name with lib.Valued and carries no @JsonValue. */
interface Valued {
    val wire: String get() = ""
}

/** Shares its simple name with lib.Coded and carries @JsonValue. */
interface Coded {
    @get:JsonValue
    val wire: String get() = ""
}

package com.example.alias.decoy

import com.example.alias.lib.Coded as Coded
import com.example.alias.lib.Valued as Valued

/** The explicit import wins over the declaration of the same name in the package: this is lib.Valued. */
enum class DecoyJson : Valued {
    A,
    B
}

/** ... and this is lib.Coded, not the decoy that carries @JsonValue. */
enum class DecoyPlain : Coded {
    DECOY_ONE,
    DECOY_TWO
}

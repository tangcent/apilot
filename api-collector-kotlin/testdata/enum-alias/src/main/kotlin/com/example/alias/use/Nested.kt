package com.example.alias.use

import com.example.alias.lib.Outer

/** The interface is nested in a class, which the file imports by its own name. */
enum class OuterJson : Outer.Valued {
    A,
    B
}

enum class OuterPlain : Outer.Coded {
    OUTER_ONE,
    OUTER_TWO
}

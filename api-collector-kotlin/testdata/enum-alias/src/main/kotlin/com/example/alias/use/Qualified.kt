package com.example.alias.use

/** The interface is written with its package. */
enum class QualifiedJson : com.example.alias.lib.Valued {
    A,
    B
}

enum class QualifiedPlain : com.example.alias.lib.Coded {
    QUALIFIED_ONE,
    QUALIFIED_TWO
}

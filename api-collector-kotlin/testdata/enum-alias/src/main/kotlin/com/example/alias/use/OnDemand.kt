package com.example.alias.use

import com.example.alias.lib.*

/** The interface is reached by an import on demand of its package. */
enum class StarJson : Valued {
    A,
    B
}

enum class StarPlain : Coded {
    STAR_ONE,
    STAR_TWO
}

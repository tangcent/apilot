package com.example.alias.use

import com.example.alias.lib.Coded
import com.example.alias.lib.Valued

/** The interface is imported by its own name. */
enum class ImportedJson : Valued {
    A,
    B
}

enum class ImportedPlain : Coded {
    IMPORTED_ONE,
    IMPORTED_TWO
}

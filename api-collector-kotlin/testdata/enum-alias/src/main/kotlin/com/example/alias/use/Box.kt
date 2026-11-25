package com.example.alias.use

import com.example.alias.lib.Coded as C
import com.example.alias.lib.Valued as V

/** The enums are nested in a class, and their interfaces are imported under an alias. */
class Box {
    enum class Kind : V {
        A,
        B
    }

    enum class Calm : C {
        BOX_ONE,
        BOX_TWO
    }
}

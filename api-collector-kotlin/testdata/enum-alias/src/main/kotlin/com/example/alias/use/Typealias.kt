package com.example.alias.use

typealias AliasedValued = com.example.alias.lib.Valued

typealias AliasedCoded = com.example.alias.lib.Coded

/** The interface is named by a typealias. */
enum class TypealiasJson : AliasedValued {
    A,
    B
}

enum class TypealiasPlain : AliasedCoded {
    TYPEALIAS_ONE,
    TYPEALIAS_TWO
}

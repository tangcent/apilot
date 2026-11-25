package com.example.alias.use

import com.example.alias.chain.Mid as M
import com.example.alias.chain.MidCoded as MC
import com.example.alias.lib.Coded as C
import com.example.alias.lib.JavaCoded as JavaC
import com.example.alias.lib.JavaValued as JavaV
import com.example.alias.lib.Outer.Coded as NC
import com.example.alias.lib.Outer.Valued as NV
import com.example.alias.lib.TypedCoded as TC
import com.example.alias.lib.TypedValued as TV
import com.example.alias.lib.Valued as V

/** The interface is imported under an alias, and the supertype list writes the alias. */
enum class RenamedJson : V {
    A,
    B
}

enum class RenamedPlain : C {
    RENAMED_ONE,
    RENAMED_TWO
}

enum class GenericJson : TV<String> {
    A,
    B
}

enum class GenericPlain : TC<String> {
    GENERIC_ONE,
    GENERIC_TWO
}

/** The interface is nested in a class, and imported under an alias itself. */
enum class NestedRenamedJson : NV {
    A,
    B
}

enum class NestedRenamedPlain : NC {
    NESTED_ONE,
    NESTED_TWO
}

/** The interface the alias names extends another one that carries @JsonValue, written in another file. */
enum class ChainJson : M {
    A,
    B
}

enum class ChainPlain : MC {
    CHAIN_ONE,
    CHAIN_TWO
}

/** The interface is written in Java. */
enum class JavaJson : JavaV {
    A,
    B
}

enum class JavaPlain : JavaC {
    JAVA_ONE,
    JAVA_TWO
}

package com.example.aliasdep

import com.example.lib.Coded
import com.example.lib.Coded as C
import com.example.lib.Valued
import com.example.lib.Valued as V
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/** The interface is imported under an alias, and the library declares it. */
enum class ByAliasJson : V {
    A,
    B
}

enum class ByAliasPlain : C {
    ALIAS_ONE,
    ALIAS_TWO
}

/** The interface is imported by its own name. */
enum class ByNameJson : Valued {
    A,
    B
}

enum class ByNamePlain : Coded {
    NAME_ONE,
    NAME_TWO
}

class AliasForm {
    var byAlias: ByAliasJson? = null
    var byAliasPlain: ByAliasPlain? = null

    var byName: ByNameJson? = null
    var byNamePlain: ByNamePlain? = null
}

@RestController
@RequestMapping("/enum-alias-dependency")
class AliasController {

    @PostMapping
    fun create(@RequestBody form: AliasForm) {
    }
}

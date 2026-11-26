package com.example.ktvalue

import com.example.ktvalue.Wire as V
import com.fasterxml.jackson.annotation.JsonValue
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

interface HasCode {
    @get:JsonValue
    val code: String
}

interface Flag {
    @get:JsonValue
    val isActive: Boolean
}

interface Wire {
    @JsonValue
    fun code(): String
}

enum class PropInherit : HasCode {
    A,
    B;

    override val code: String
        get() = "x"
}

enum class PropOffGet : HasCode {
    A,
    B;

    @get:JsonValue(false)
    override val code: String
        get() = "x"
}

enum class PropOffField : HasCode {
    A,
    B;

    @field:JsonValue(false)
    override val code: String = "x"
}

enum class PropOffPlain : HasCode {
    A,
    B;

    @JsonValue(false)
    override val code: String = "x"
}

enum class CtorPlain(@JsonValue val code: String) {
    A("a"),
    B("b")
}

enum class CtorGet(@get:JsonValue val code: String) {
    A("a"),
    B("b")
}

enum class SetOnly {
    A,
    B;

    @set:JsonValue
    var code: String = "x"
}

enum class FlagInherit : Flag {
    A,
    B;

    override val isActive: Boolean
        get() = true
}

enum class FlagOff : Flag {
    A,
    B;

    @get:JsonValue(false)
    override val isActive: Boolean
        get() = true
}

enum class AliasedOn : V {
    A,
    B;

    override fun code(): String = "x"
}

enum class AliasedOff : V {
    A,
    B;

    @JsonValue(false)
    override fun code(): String = "x"
}

class KtForm {
    var propInherit: PropInherit? = null
    var propOffGet: PropOffGet? = null
    var propOffField: PropOffField? = null
    var propOffPlain: PropOffPlain? = null
    var ctorPlain: CtorPlain? = null
    var ctorGet: CtorGet? = null
    var setOnly: SetOnly? = null
    var flagInherit: FlagInherit? = null
    var flagOff: FlagOff? = null
    var aliasedOn: AliasedOn? = null
    var aliasedOff: AliasedOff? = null
}

@RestController
@RequestMapping("/ktvalue")
class KtController {
    @PostMapping
    fun create(@RequestBody form: KtForm) {
    }

    @GetMapping
    fun page(query: KtForm) {
    }
}

package com.example.cross

import com.fasterxml.jackson.annotation.JsonValue
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

interface KtCoded {
    @get:JsonValue
    val code: String
}

enum class KtOn : JavaCoded {
    A,
    B;

    override fun getCode(): String = "x"
}

enum class KtOff : JavaCoded {
    A,
    B;

    @JsonValue(false)
    override fun getCode(): String = "x"
}

class CrossForm {
    var ktOn: KtOn? = null
    var ktOff: KtOff? = null
    var javaOn: JavaOn? = null
    var javaOff: JavaOff? = null
}

@RestController
@RequestMapping("/cross")
class CrossController {
    @PostMapping
    fun create(@RequestBody form: CrossForm) {
    }
}

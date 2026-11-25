package com.example.builtin.shadow

import com.example.builtin.lib.Box
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/** The package declares Int, ByteArray and Number, which hide Kotlin's. */
class Int(val digits: kotlin.String?)

class ByteArray(val raw: kotlin.String?)

class Number(val digits: kotlin.String?)

class ShadowForm(
    val count: Int,
    val bytes: ByteArray,
    val weight: Number,
    val total: Long,
    val label: String,
)

@RestController
@RequestMapping("/shadow")
class ShadowController {

    @PostMapping("/save")
    fun save(@RequestBody form: ShadowForm): ShadowForm = form

    @GetMapping("/box")
    fun box(): Box<Int> = TODO()
}

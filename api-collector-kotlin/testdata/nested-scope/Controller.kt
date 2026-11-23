package com.example.kcnest

import org.springframework.web.bind.annotation.DeleteMapping
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/** Int, Number and Long are Kotlin's here: the classes of that name are nested or private elsewhere. */
class Form(
    val count: Int,
    val weight: Number,
    val counts: List<Int>,
    val total: Long,
)

@RestController
@RequestMapping("/nest")
class Controller {

    @PostMapping("/save")
    fun save(@RequestBody form: Form): Form = form

    @GetMapping("/count")
    fun count(): Int = TODO()

    @DeleteMapping("/clear")
    fun clear() {
    }

    /** The Unit nested in Holder is not in scope here. */
    @GetMapping("/unit")
    fun unit(): Unit {
    }

    @GetMapping("/private-user")
    fun privateUser(): PrivateUser = TODO()
}

package com.example.builtin.lib

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody

/** Classes of the library that bear the names of Kotlin's types. */
class Int(val shadow: kotlin.Int?)

class Integer(val shadow: kotlin.Int?)

class String(val shadow: kotlin.Int?)

class Box<T>(
    val data: T?,
    val list: List<T>?,
    val byKey: Map<kotlin.String, T>?,
)

class Wrapper<U>(
    val data: U?,
)

class Outer<T>(
    val wrapped: Wrapper<T>?,
    val wrappedList: Wrapper<List<T>>?,
)

/** A generic controller of the library, in a package that declares Int and String. */
abstract class BaseController<T> {

    @GetMapping("/one")
    fun one(): T = TODO()

    @GetMapping("/all")
    fun all(): List<T> = TODO()

    @PostMapping("/save")
    fun save(@RequestBody body: T): T = body
}

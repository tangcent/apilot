package com.example.typearg.lib

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody

/** The library's own Item, which shares its simple name with the application's. */
class Item(
    val code: String?,
)

class Box<T>(
    val data: T?,
    val list: List<T>?,
    val array: Array<T>?,
    val byKey: Map<String, T>?,
)

class Wrapper<U>(
    val data: U?,
)

class Outer<T>(
    /** The parameter is handed on to the next generic class. */
    val wrapped: Wrapper<T>?,
    val wrappedList: Wrapper<List<T>>?,
    val wrappedArray: Wrapper<Array<T>>?,
)

class Holder(
    /** Written in full: the application's class, though the library has an Item too. */
    val fixedList: List<com.example.typearg.dto.Item>?,
    val fixedBox: Box<com.example.typearg.dto.Item>?,
    /** Written simply: the library's own class. */
    val own: Item?,
)

/** A generic controller of the library, in a package that has its own Item. */
abstract class BaseController<T> {

    @GetMapping("/one")
    fun one(): T = TODO()

    @GetMapping("/all")
    fun all(): List<T> = TODO()

    @PostMapping("/save")
    fun save(@RequestBody body: T): T = body

    @GetMapping("/search")
    fun search(query: T): List<T> = TODO()
}

/** Names its own parameter T, like the controller it extends. */
abstract class MidController<T> : BaseController<List<T>>() {

    @GetMapping("/mid")
    fun mid(): T = TODO()
}

package com.example.builtin.own

import com.example.builtin.lib.BaseController
import com.example.builtin.lib.Box
import com.example.builtin.lib.Outer
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/** The package declares Integer, Number and Unit; Int, Long and String are Kotlin's. */
class Integer(val digits: kotlin.String?)

class Number(val digits: kotlin.String?)

enum class Unit { PIECE, BOX }

class CountForm(
    val count: Int,
    val total: Long,
    val name: String,
    val flag: Boolean,
    val counts: List<Int>,
    val bytes: ByteArray?,
    /** Integer is the class of this package, not the one Int stands for. */
    val boxed: Integer,
    val weight: Number,
    val unit: Unit,
)

@RestController
@RequestMapping("/own")
class OwnController {

    @PostMapping("/save")
    fun save(@RequestBody form: CountForm): CountForm = form

    @GetMapping("/search")
    fun search(form: CountForm): List<CountForm> = TODO()

    /** Unit is the enum of this package, so the function returns a value. */
    @GetMapping("/unit")
    fun unit(): Unit = TODO()

    @GetMapping("/box-int")
    fun boxInt(): Box<Int> = TODO()

    @GetMapping("/box-string")
    fun boxString(): Box<String> = TODO()

    @GetMapping("/box-integer")
    fun boxInteger(): Box<Integer> = TODO()

    @GetMapping("/box-list")
    fun boxList(): Box<List<Int>> = TODO()

    @GetMapping("/outer-int")
    fun outerInt(): Outer<Int> = TODO()
}

@RestController
@RequestMapping("/own-items")
class OwnItemController : BaseController<Int>()

@RestController
@RequestMapping("/own-names")
class OwnNameController : BaseController<String>()

package com.example.arrayfield

import javax.validation.constraints.NotEmpty
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

class ArrayFieldItem(
    val name: String?,
)

class ArrayFieldForm(
    val names: Array<String>?,
    val ids: IntArray?,
    val longs: Array<Long>?,
    val items: Array<ArrayFieldItem>?,
    val groups: Array<List<String>>?,
    val grid: Array<IntArray>?,
    /** A name qualified with its package is the same type as the simple one. */
    val day: java.time.LocalDate?,
    val amounts: Array<java.math.BigDecimal>?,
    @field:NotEmpty
    val codes: Array<String>?,
)

@RestController
@RequestMapping("/array-fields")
class ArrayFieldController {

    @PostMapping("/save")
    fun save(@RequestBody form: ArrayFieldForm): ArrayFieldForm = form

    @PostMapping("/names")
    fun names(@RequestBody names: Array<String>): Array<String> = names

    @GetMapping("/items")
    fun items(): Array<ArrayFieldItem> = emptyArray()
}

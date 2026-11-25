package com.example.builtin.star

import com.example.builtin.lib.Box
import com.example.builtin.shadow.*
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/** A star-import wins over the packages every file imports. */
class StarForm(
    val count: Int,
    val bytes: ByteArray,
    val weight: Number,
    val total: Long,
    val label: String,
)

@RestController
@RequestMapping("/star")
class StarController {

    @PostMapping("/save")
    fun save(@RequestBody form: StarForm): StarForm = form

    @GetMapping("/box")
    fun box(): Box<Int> = TODO()
}

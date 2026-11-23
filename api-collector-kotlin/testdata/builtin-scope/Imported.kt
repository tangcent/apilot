package com.example.builtin.imported

import com.example.builtin.lib.Box
import com.example.builtin.shadow.ByteArray
import com.example.builtin.shadow.Int
import com.example.builtin.shadow.Number
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/** Int, ByteArray and Number are imported by name from a package that declares them. */
class ImportedForm(
    val count: Int,
    val bytes: ByteArray,
    val weight: Number,
    val total: Long,
    val label: String,
)

@RestController
@RequestMapping("/imported")
class ImportedController {

    @PostMapping("/save")
    fun save(@RequestBody form: ImportedForm): ImportedForm = form

    @GetMapping("/box")
    fun box(): Box<Int> = TODO()
}

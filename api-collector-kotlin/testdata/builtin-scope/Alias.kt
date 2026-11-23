package com.example.builtin.alias

import com.example.builtin.shadow.ByteArray as Bytes
import com.example.builtin.shadow.Number as Int
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/** A renaming import binds only the new name: here Int is a class of the project. */
class AliasForm(
    val count: Int,
    val bytes: Bytes,
    val total: Long,
)

@RestController
@RequestMapping("/alias")
class AliasController {

    @PostMapping("/save")
    fun save(@RequestBody form: AliasForm): AliasForm = form
}

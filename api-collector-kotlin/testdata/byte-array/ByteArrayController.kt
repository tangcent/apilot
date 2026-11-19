package com.example.bytearray

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

class ByteArrayForm(
    /** Written as one Base64 string. */
    val data: ByteArray?,
    val chunks: Array<ByteArray>?,
    val blobs: List<ByteArray>?,
    /** Boxed bytes are numbers, one by one. */
    val boxed: Array<Byte>?,
    val numbers: List<Byte>?,
    val single: Byte?,
    val ints: IntArray?,
)

class ByteArrayQuery(
    val data: ByteArray?,
    val boxed: Array<Byte>?,
)

@RestController
@RequestMapping("/byte-arrays")
class ByteArrayController {

    @PostMapping("/save")
    fun save(@RequestBody form: ByteArrayForm): ByteArrayForm = form

    @PostMapping("/blob")
    fun blob(@RequestBody blob: ByteArray): ByteArray = blob

    @GetMapping("/query")
    fun query(@RequestParam data: ByteArray, @RequestParam boxed: Array<Byte>): String = ""

    @GetMapping("/bound")
    fun bound(query: ByteArrayQuery): String = ""
}

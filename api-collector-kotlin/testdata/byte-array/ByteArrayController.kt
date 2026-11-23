package com.example.bytearray

import org.springframework.http.ResponseEntity
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

    @PostMapping("/outer")
    fun outer(@RequestBody body: ByteArrayOuter<ByteArray?>): ByteArrayOuter<ByteArray> = ByteArrayOuter(null, null, null)

    @PostMapping("/outer-boxed")
    fun outerBoxed(@RequestBody body: ByteArrayOuter<Array<Byte>>): ByteArrayOuter<Array<Byte>> = body

    @PostMapping("/outer-chunks")
    fun outerChunks(@RequestBody body: ByteArrayOuter<Array<ByteArray>>): ByteArrayOuter<Array<ByteArray>> = body

    @PostMapping("/outer-list")
    fun outerList(@RequestBody body: ByteArrayOuter<List<ByteArray>>): ResponseEntity<ByteArrayOuter<List<ByteArray>>>? = null

    @PostMapping("/outer-wildcard")
    fun outerWildcard(@RequestBody body: ByteArrayOuter<out ByteArray>): ByteArrayOuter<out ByteArray> = body

    @PostMapping("/sub")
    fun sub(@RequestBody body: ByteArraySub): ByteArraySub = body

    @PostMapping("/sub-boxed")
    fun subBoxed(@RequestBody body: ByteArrayBoxedSub): ByteArrayBoxedSub = body

    @PostMapping("/open-sub")
    fun openSub(@RequestBody body: ByteArrayOpenSub<ByteArray>): ByteArrayOpenSub<ByteArray> = body

    @PostMapping("/by-name")
    fun byName(@RequestBody body: Map<String, ByteArrayWrapper<ByteArray>>): Map<String, ByteArrayWrapper<ByteArray>> = body

    @GetMapping("/query")
    fun query(@RequestParam data: ByteArray, @RequestParam boxed: Array<Byte>): String = ""

    @GetMapping("/bound-outer")
    fun boundOuter(query: ByteArrayOuter<ByteArray>): String = ""

    @GetMapping("/bound")
    fun bound(query: ByteArrayQuery): String = ""
}

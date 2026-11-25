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
    /** A CharArray is written as one string of its characters. */
    val letters: CharArray?,
    val lines: Array<CharArray>?,
    val words: List<CharArray>?,
    val lettersByName: Map<String, CharArray>?,
    /** Boxed characters are strings, one by one. */
    val boxedLetters: Array<Char>?,
    val letterList: List<Char>?,
    val letter: Char?,
)

class ByteArrayQuery(
    val data: ByteArray?,
    val boxed: Array<Byte>?,
    val letters: CharArray?,
    val boxedLetters: Array<Char>?,
)

@RestController
@RequestMapping("/byte-arrays")
class ByteArrayController {

    @PostMapping("/save")
    fun save(@RequestBody form: ByteArrayForm): ByteArrayForm = form

    @PostMapping("/blob")
    fun blob(@RequestBody blob: ByteArray): ByteArray = blob

    @PostMapping("/text")
    fun text(@RequestBody text: CharArray): CharArray = text

    @PostMapping("/outer-chars")
    fun outerChars(@RequestBody body: ByteArrayOuter<CharArray>): ByteArrayOuter<CharArray> = body

    @PostMapping("/outer-boxed-chars")
    fun outerBoxedChars(@RequestBody body: ByteArrayOuter<Array<Char>>): ByteArrayOuter<Array<Char>> = body

    @PostMapping("/char-sub")
    fun charSub(@RequestBody body: ByteArrayCharSub): ByteArrayCharSub = body

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
    fun query(
        @RequestParam data: ByteArray,
        @RequestParam boxed: Array<Byte>,
        @RequestParam letters: CharArray,
        @RequestParam boxedLetters: Array<Char>,
    ): String = ""

    @GetMapping("/bound-outer")
    fun boundOuter(query: ByteArrayOuter<ByteArray>): String = ""

    @GetMapping("/bound-outer-chars")
    fun boundOuterChars(query: ByteArrayOuter<CharArray>): String = ""

    @GetMapping("/bound")
    fun bound(query: ByteArrayQuery): String = ""

    // A vararg of a primitive type is the primitive array, which is what the
    // explicit declaration beside each is; a vararg of a nullable element is an
    // Array of boxed values.

    @PostMapping("/explicit/bytes")
    fun explicitBytes(@RequestBody bytes: ByteArray): ByteArray = bytes

    @PostMapping("/vararg/bytes")
    fun varargBytes(@RequestBody vararg bytes: Byte): ByteArray = bytes

    @PostMapping("/explicit/chars")
    fun explicitChars(@RequestBody chars: CharArray): CharArray = chars

    @PostMapping("/vararg/chars")
    fun varargChars(@RequestBody vararg chars: Char): CharArray = chars

    @PostMapping("/explicit/boxed-bytes")
    fun explicitBoxedBytes(@RequestBody bytes: Array<Byte?>): Array<Byte?> = bytes

    @PostMapping("/vararg/boxed-bytes")
    fun varargBoxedBytes(@RequestBody vararg bytes: Byte?): Array<Byte?> = arrayOf(*bytes)

    @PostMapping("/explicit/boxed-chars")
    fun explicitBoxedChars(@RequestBody chars: Array<Char?>): Array<Char?> = chars

    @PostMapping("/vararg/boxed-chars")
    fun varargBoxedChars(@RequestBody vararg chars: Char?): Array<Char?> = arrayOf(*chars)

    @PostMapping("/explicit/ints")
    fun explicitInts(@RequestBody ints: IntArray): IntArray = ints

    @PostMapping("/vararg/ints")
    fun varargInts(@RequestBody vararg ints: Int): IntArray = ints

    @GetMapping("/explicit-query/bytes")
    fun explicitQueryBytes(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/bytes")
    fun varargQueryBytes(@RequestParam vararg data: Byte): String = ""

    @GetMapping("/explicit-query/chars")
    fun explicitQueryChars(@RequestParam data: CharArray): String = ""

    @GetMapping("/vararg-query/chars")
    fun varargQueryChars(@RequestParam vararg data: Char): String = ""

    @GetMapping("/explicit-query/boxed-bytes")
    fun explicitQueryBoxedBytes(@RequestParam data: Array<Byte?>): String = ""

    @GetMapping("/vararg-query/boxed-bytes")
    fun varargQueryBoxedBytes(@RequestParam vararg data: Byte?): String = ""

    @GetMapping("/explicit-query/boxed-chars")
    fun explicitQueryBoxedChars(@RequestParam data: Array<Char?>): String = ""

    @GetMapping("/vararg-query/boxed-chars")
    fun varargQueryBoxedChars(@RequestParam vararg data: Char?): String = ""
}

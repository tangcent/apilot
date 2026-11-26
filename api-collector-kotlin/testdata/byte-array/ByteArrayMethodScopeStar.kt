package com.example.bytearray

import com.example.bytearray.alias.*
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

// A star import binds the aliases. A type parameter of the function is inside
// that import and hides only its own name.

@RestController
@RequestMapping("/byte-scope-star")
class ByteArrayMethodScopeStarController {

    @PostMapping("/explicit/star-octet")
    fun <StarOctet> explicitStarOctet(@RequestBody body: Array<StarOctet>): Array<StarOctet> = body

    @PostMapping("/vararg/star-octet")
    fun <StarOctet> varargStarOctet(@RequestBody vararg data: StarOctet): Array<out StarOctet> = data

    @GetMapping("/explicit-query/star-octet")
    fun <StarOctet> explicitQueryStarOctet(@RequestParam data: Array<StarOctet>): String = ""

    @GetMapping("/vararg-query/star-octet")
    fun <StarOctet> varargQueryStarOctet(@RequestParam vararg data: StarOctet): String = ""

    @PostMapping("/explicit/star-kept")
    fun <T> explicitStarKept(@RequestBody body: ByteArray): ByteArray = body

    @PostMapping("/vararg/star-kept")
    fun <T> varargStarKept(@RequestBody vararg data: StarOctet): ByteArray = data

    @GetMapping("/explicit-query/star-kept")
    fun <T> explicitQueryStarKept(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/star-kept")
    fun <T> varargQueryStarKept(@RequestParam vararg data: StarOctet): String = ""

    @PostMapping("/explicit/star-letter")
    fun <StarLetter> explicitStarLetter(@RequestBody body: Array<StarLetter>): Array<StarLetter> = body

    @PostMapping("/vararg/star-letter")
    fun <StarLetter> varargStarLetter(@RequestBody vararg data: StarLetter): Array<out StarLetter> = data

    @GetMapping("/explicit-query/star-letter")
    fun <StarLetter> explicitQueryStarLetter(@RequestParam data: Array<StarLetter>): String = ""

    @GetMapping("/vararg-query/star-letter")
    fun <StarLetter> varargQueryStarLetter(@RequestParam vararg data: StarLetter): String = ""

    @PostMapping("/explicit/star-other")
    fun <StarOctet> explicitStarOther(@RequestBody body: CharArray): CharArray = body

    @PostMapping("/vararg/star-other")
    fun <StarOctet> varargStarOther(@RequestBody vararg data: StarLetter): CharArray = data

    @GetMapping("/explicit-query/star-other")
    fun <StarOctet> explicitQueryStarOther(@RequestParam data: CharArray): String = ""

    @GetMapping("/vararg-query/star-other")
    fun <StarOctet> varargQueryStarOther(@RequestParam vararg data: StarLetter): String = ""
}

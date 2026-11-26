package com.example.bytearray

import com.example.bytearray.alias.PlainOctet
import com.example.bytearray.alias.RenamedOctet as Renamed
import com.example.bytearray.shadow.Byte
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

// Byte is a class of the project here. A type parameter of the function is
// inside that import; an alias is still read in the file that declares it.

@RestController
@RequestMapping("/byte-scope-import")
class ByteArrayMethodScopeImportController {

    @PostMapping("/explicit/project-byte")
    fun <Byte> explicitProjectByte(@RequestBody body: Array<Byte>): Array<Byte> = body

    @PostMapping("/vararg/project-byte")
    fun <Byte> varargProjectByte(@RequestBody vararg data: Byte): Array<out Byte> = data

    @GetMapping("/explicit-query/project-byte")
    fun <Byte> explicitQueryProjectByte(@RequestParam data: Array<Byte>): String = ""

    @GetMapping("/vararg-query/project-byte")
    fun <Byte> varargQueryProjectByte(@RequestParam vararg data: Byte): String = ""

    @PostMapping("/explicit/project-plain")
    fun explicitProjectPlain(@RequestBody body: Array<Byte>): Array<Byte> = body

    @PostMapping("/vararg/project-plain")
    fun varargProjectPlain(@RequestBody vararg data: Byte): Array<out Byte> = data

    @GetMapping("/explicit-query/project-plain")
    fun explicitQueryProjectPlain(@RequestParam data: Array<Byte>): String = ""

    @GetMapping("/vararg-query/project-plain")
    fun varargQueryProjectPlain(@RequestParam vararg data: Byte): String = ""

    @PostMapping("/explicit/plain-import")
    fun <PlainOctet> explicitPlainImport(@RequestBody body: Array<PlainOctet>): Array<PlainOctet> = body

    @PostMapping("/vararg/plain-import")
    fun <PlainOctet> varargPlainImport(@RequestBody vararg data: PlainOctet): Array<out PlainOctet> = data

    @GetMapping("/explicit-query/plain-import")
    fun <PlainOctet> explicitQueryPlainImport(@RequestParam data: Array<PlainOctet>): String = ""

    @GetMapping("/vararg-query/plain-import")
    fun <PlainOctet> varargQueryPlainImport(@RequestParam vararg data: PlainOctet): String = ""

    @PostMapping("/explicit/renamed")
    fun <Renamed> explicitRenamed(@RequestBody body: Array<Renamed>): Array<Renamed> = body

    @PostMapping("/vararg/renamed")
    fun <Renamed> varargRenamed(@RequestBody vararg data: Renamed): Array<out Renamed> = data

    @GetMapping("/explicit-query/renamed")
    fun <Renamed> explicitQueryRenamed(@RequestParam data: Array<Renamed>): String = ""

    @GetMapping("/vararg-query/renamed")
    fun <Renamed> varargQueryRenamed(@RequestParam vararg data: Renamed): String = ""

    @PostMapping("/explicit/renamed-kept")
    fun <T> explicitRenamedKept(@RequestBody body: ByteArray): ByteArray = body

    @PostMapping("/vararg/renamed-kept")
    fun <T> varargRenamedKept(@RequestBody vararg data: Renamed): ByteArray = data

    @GetMapping("/explicit-query/renamed-kept")
    fun <T> explicitQueryRenamedKept(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/renamed-kept")
    fun <T> varargQueryRenamedKept(@RequestParam vararg data: Renamed): String = ""
}

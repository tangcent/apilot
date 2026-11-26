package com.example.bytearray

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

// A type parameter of the function hides a class of the same name, including
// Byte and Char of the standard library. The vararg stays an Array of that
// parameter, which is what the explicit declaration beside it is. A function
// that does not declare the parameter is unchanged.

@RestController
@RequestMapping("/byte-scope")
class ByteArrayMethodScopeController {

    @PostMapping("/explicit/byte")
    fun <Byte> explicitByte(@RequestBody body: Array<Byte>): Array<Byte> = body

    @PostMapping("/vararg/byte")
    fun <Byte> varargByte(@RequestBody vararg data: Byte): Array<out Byte> = data

    @GetMapping("/explicit-query/byte")
    fun <Byte> explicitQueryByte(@RequestParam data: Array<Byte>): String = ""

    @GetMapping("/vararg-query/byte")
    fun <Byte> varargQueryByte(@RequestParam vararg data: Byte): String = ""

    @PostMapping("/explicit/char")
    fun <Char> explicitChar(@RequestBody body: Array<Char>): Array<Char> = body

    @PostMapping("/vararg/char")
    fun <Char> varargChar(@RequestBody vararg data: Char): Array<out Char> = data

    @GetMapping("/explicit-query/char")
    fun <Char> explicitQueryChar(@RequestParam data: Array<Char>): String = ""

    @GetMapping("/vararg-query/char")
    fun <Char> varargQueryChar(@RequestParam vararg data: Char): String = ""

    @PostMapping("/explicit/int")
    fun <Int> explicitInt(@RequestBody body: Array<Int>): Array<Int> = body

    @PostMapping("/vararg/int")
    fun <Int> varargInt(@RequestBody vararg data: Int): Array<out Int> = data

    @GetMapping("/explicit-query/int")
    fun <Int> explicitQueryInt(@RequestParam data: Array<Int>): String = ""

    @GetMapping("/vararg-query/int")
    fun <Int> varargQueryInt(@RequestParam vararg data: Int): String = ""

    @PostMapping("/explicit/ubyte")
    fun <UByte> explicitUbyte(@RequestBody body: Array<UByte>): Array<UByte> = body

    @PostMapping("/vararg/ubyte")
    fun <UByte> varargUbyte(@RequestBody vararg data: UByte): Array<out UByte> = data

    @GetMapping("/explicit-query/ubyte")
    fun <UByte> explicitQueryUbyte(@RequestParam data: Array<UByte>): String = ""

    @GetMapping("/vararg-query/ubyte")
    fun <UByte> varargQueryUbyte(@RequestParam vararg data: UByte): String = ""

    @PostMapping("/explicit/bounded")
    fun <Byte : Number> explicitBounded(@RequestBody body: Array<Byte>): Array<Byte> = body

    @PostMapping("/vararg/bounded")
    fun <Byte : Number> varargBounded(@RequestBody vararg data: Byte): Array<out Byte> = data

    @GetMapping("/explicit-query/bounded")
    fun <Byte : Number> explicitQueryBounded(@RequestParam data: Array<Byte>): String = ""

    @GetMapping("/vararg-query/bounded")
    fun <Byte : Number> varargQueryBounded(@RequestParam vararg data: Byte): String = ""

    @PostMapping("/explicit/where")
    fun <Byte> explicitWhere(@RequestBody body: Array<Byte>): Array<Byte> where Byte : Number = body

    @PostMapping("/vararg/where")
    fun <Byte> varargWhere(@RequestBody vararg data: Byte): Array<out Byte> where Byte : Number = data

    @GetMapping("/explicit-query/where")
    fun <Byte> explicitQueryWhere(@RequestParam data: Array<Byte>): String where Byte : Number = ""

    @GetMapping("/vararg-query/where")
    fun <Byte> varargQueryWhere(@RequestParam vararg data: Byte): String where Byte : Number = ""

    @PostMapping("/explicit/reified")
    inline fun <reified Byte> explicitReified(@RequestBody body: Array<Byte>): Array<Byte> = body

    @PostMapping("/vararg/reified")
    inline fun <reified Byte> varargReified(@RequestBody vararg data: Byte): Array<out Byte> = data

    @GetMapping("/explicit-query/reified")
    inline fun <reified Byte> explicitQueryReified(@RequestParam data: Array<Byte>): String = ""

    @GetMapping("/vararg-query/reified")
    inline fun <reified Byte> varargQueryReified(@RequestParam vararg data: Byte): String = ""

    @PostMapping("/explicit/multi")
    fun <T : CharSequence, Byte> explicitMulti(@RequestBody body: Array<Byte>): Array<Byte> = body

    @PostMapping("/vararg/multi")
    fun <T : CharSequence, Byte> varargMulti(@RequestBody vararg data: Byte): Array<out Byte> = data

    @GetMapping("/explicit-query/multi")
    fun <T : CharSequence, Byte> explicitQueryMulti(@RequestParam data: Array<Byte>): String = ""

    @GetMapping("/vararg-query/multi")
    fun <T : CharSequence, Byte> varargQueryMulti(@RequestParam vararg data: Byte): String = ""

    // The parameter hides the alias of the same name, not a different one, and
    // not a name written in full.

    @PostMapping("/explicit/shadow-alias")
    fun <Octet> explicitShadowAlias(@RequestBody body: Array<Octet>): Array<Octet> = body

    @PostMapping("/vararg/shadow-alias")
    fun <Octet> varargShadowAlias(@RequestBody vararg data: Octet): Array<out Octet> = data

    @GetMapping("/explicit-query/shadow-alias")
    fun <Octet> explicitQueryShadowAlias(@RequestParam data: Array<Octet>): String = ""

    @GetMapping("/vararg-query/shadow-alias")
    fun <Octet> varargQueryShadowAlias(@RequestParam vararg data: Octet): String = ""

    @PostMapping("/explicit/alias-target")
    fun <Byte> explicitAliasTarget(@RequestBody body: ByteArray): ByteArray = body

    @PostMapping("/vararg/alias-target")
    fun <Byte> varargAliasTarget(@RequestBody vararg data: Octet): ByteArray = data

    @GetMapping("/explicit-query/alias-target")
    fun <Byte> explicitQueryAliasTarget(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/alias-target")
    fun <Byte> varargQueryAliasTarget(@RequestParam vararg data: Octet): String = ""

    @PostMapping("/explicit/qualified")
    fun <Byte> explicitQualified(@RequestBody body: ByteArray): ByteArray = body

    @PostMapping("/vararg/qualified")
    fun <Byte> varargQualified(@RequestBody vararg data: kotlin.Byte): ByteArray = data

    @GetMapping("/explicit-query/qualified")
    fun <Byte> explicitQueryQualified(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/qualified")
    fun <Byte> varargQueryQualified(@RequestParam vararg data: kotlin.Byte): String = ""

    @PostMapping("/explicit/beside")
    fun <Byte> explicitBeside(@RequestBody body: CharArray): CharArray = body

    @PostMapping("/vararg/beside")
    fun <Byte> varargBeside(@RequestBody vararg data: Char): CharArray = data

    @GetMapping("/explicit-query/beside")
    fun <Byte> explicitQueryBeside(@RequestParam data: CharArray): String = ""

    @GetMapping("/vararg-query/beside")
    fun <Byte> varargQueryBeside(@RequestParam vararg data: Char): String = ""

    @PostMapping("/leak/plain")
    fun leakPlain(@RequestBody vararg data: Byte): ByteArray = data

    @PostMapping("/plain-param/typed")
    fun <Byte> typedParam(@RequestBody body: Byte): Byte = body

    @PostMapping("/plain-param/plain")
    fun plainParam(@RequestBody body: Byte): Byte = body

    @PostMapping("/blob")
    fun blob(@RequestBody body: Blob): Blob = body

    @PostMapping("/blob-field")
    fun blobField(@RequestBody form: ByteScopeBlobForm): ByteScopeBlobForm = form
}

class ByteScopeBlobForm(val data: Blob?)

@RestController
@RequestMapping("/byte-scope/class")
class ByteScopeClass<Byte> {

    @PostMapping("/explicit")
    fun explicit(@RequestBody body: Array<Byte>): Array<Byte> = body

    @PostMapping("/vararg")
    fun take(@RequestBody vararg data: Byte): Array<out Byte> = data
}

class ByteScopeHolder<Byte> {

    @RestController
    @RequestMapping("/byte-scope/inner")
    inner class ByteScopeInner {

        @PostMapping("/explicit")
        fun explicit(@RequestBody body: Array<Byte>): Array<Byte> = body

        @PostMapping("/vararg")
        fun take(@RequestBody vararg data: Byte): Array<out Byte> = data
    }
}

class ByteScopeNestedHolder<Byte> {

    @RestController
    @RequestMapping("/byte-scope/nested")
    class ByteScopeNested {

        @PostMapping("/vararg")
        fun take(@RequestBody vararg data: Byte): ByteArray = data
    }
}

class ByteScopeTypedVararg<Byte>(vararg val data: Byte)

class ByteScopeTypedExplicit<Byte>(val data: Array<Byte>)

@RestController
@RequestMapping("/byte-scope/ctor")
class ByteScopeCtorController {

    @PostMapping("/explicit")
    fun explicit(@RequestBody form: ByteScopeTypedExplicit<String>): ByteScopeTypedExplicit<String> = form

    @PostMapping("/vararg")
    fun take(@RequestBody form: ByteScopeTypedVararg<String>): ByteScopeTypedVararg<String> = form
}

package com.example.bytearray

import com.example.bytearray.alias.OtherMaybe
import com.example.bytearray.alias.PlainOctet
import com.example.bytearray.alias.RenamedOctet as Aliased
import com.example.bytearray.alias.StarLetter
import com.example.bytearray.alias.StarOctet
import com.example.bytearray.shadow.Byte as ShadowByte
import com.example.bytearray.shadow.Shadowed
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

typealias Octet = Byte
typealias Letter = Char
typealias Whole = Int
typealias UOctet = UByte
typealias MaybeByte = Byte?
typealias MaybeChar = Char?
typealias Octet2 = Octet
typealias Octet3 = Octet2
typealias Bottom = Byte
typealias Middle = Bottom?
typealias Top = Middle
typealias QualifiedByte = kotlin.Byte
typealias Const<T> = Byte
typealias MaybeConst<T> = Byte?
typealias Text = String
typealias Blob = ByteArray
private typealias Secret = Byte

// A typealias of a primitive type is the type it stands for: `vararg Octet` is
// the ByteArray that `vararg Byte` is, and one whose target is nullable at any
// level is an Array of boxed values, as `vararg Octet?` is. Each handler has an
// explicit twin, declared with the type Kotlin gives the vararg.
@RestController
@RequestMapping("/byte-aliases")
class ByteArrayAliasController {

    @PostMapping("/explicit/octet")
    fun explicitOctet(@RequestBody x: ByteArray): ByteArray = x

    @PostMapping("/vararg/octet")
    fun varargOctet(@RequestBody vararg x: Octet): ByteArray = x

    @GetMapping("/explicit-query/octet")
    fun explicitQueryOctet(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/octet")
    fun varargQueryOctet(@RequestParam vararg data: Octet): String = ""

    @PostMapping("/explicit/letter")
    fun explicitLetter(@RequestBody x: CharArray): CharArray = x

    @PostMapping("/vararg/letter")
    fun varargLetter(@RequestBody vararg x: Letter): CharArray = x

    @GetMapping("/explicit-query/letter")
    fun explicitQueryLetter(@RequestParam data: CharArray): String = ""

    @GetMapping("/vararg-query/letter")
    fun varargQueryLetter(@RequestParam vararg data: Letter): String = ""

    @PostMapping("/explicit/whole")
    fun explicitWhole(@RequestBody x: IntArray): IntArray = x

    @PostMapping("/vararg/whole")
    fun varargWhole(@RequestBody vararg x: Whole): IntArray = x

    @GetMapping("/explicit-query/whole")
    fun explicitQueryWhole(@RequestParam data: IntArray): String = ""

    @GetMapping("/vararg-query/whole")
    fun varargQueryWhole(@RequestParam vararg data: Whole): String = ""

    @PostMapping("/explicit/ubyte")
    fun explicitUbyte(@RequestBody x: UByteArray): UByteArray = x

    @PostMapping("/vararg/ubyte")
    fun varargUbyte(@RequestBody vararg x: UOctet): UByteArray = x

    @GetMapping("/explicit-query/ubyte")
    fun explicitQueryUbyte(@RequestParam data: UByteArray): String = ""

    @GetMapping("/vararg-query/ubyte")
    fun varargQueryUbyte(@RequestParam vararg data: UOctet): String = ""

    @PostMapping("/explicit/maybe-byte")
    fun explicitMaybeByte(@RequestBody x: Array<Byte?>): Array<Byte?> = x

    @PostMapping("/vararg/maybe-byte")
    fun varargMaybeByte(@RequestBody vararg x: MaybeByte): Array<Byte?> = arrayOf(*x)

    @GetMapping("/explicit-query/maybe-byte")
    fun explicitQueryMaybeByte(@RequestParam data: Array<Byte?>): String = ""

    @GetMapping("/vararg-query/maybe-byte")
    fun varargQueryMaybeByte(@RequestParam vararg data: MaybeByte): String = ""

    @PostMapping("/explicit/maybe-char")
    fun explicitMaybeChar(@RequestBody x: Array<Char?>): Array<Char?> = x

    @PostMapping("/vararg/maybe-char")
    fun varargMaybeChar(@RequestBody vararg x: MaybeChar): Array<Char?> = arrayOf(*x)

    @GetMapping("/explicit-query/maybe-char")
    fun explicitQueryMaybeChar(@RequestParam data: Array<Char?>): String = ""

    @GetMapping("/vararg-query/maybe-char")
    fun varargQueryMaybeChar(@RequestParam vararg data: MaybeChar): String = ""

    @PostMapping("/explicit/octet-nullable")
    fun explicitOctetNullable(@RequestBody x: Array<Byte?>): Array<Byte?> = x

    @PostMapping("/vararg/octet-nullable")
    fun varargOctetNullable(@RequestBody vararg x: Octet?): Array<Byte?> = arrayOf(*x)

    @GetMapping("/explicit-query/octet-nullable")
    fun explicitQueryOctetNullable(@RequestParam data: Array<Byte?>): String = ""

    @GetMapping("/vararg-query/octet-nullable")
    fun varargQueryOctetNullable(@RequestParam vararg data: Octet?): String = ""

    @PostMapping("/explicit/chain")
    fun explicitChain(@RequestBody x: ByteArray): ByteArray = x

    @PostMapping("/vararg/chain")
    fun varargChain(@RequestBody vararg x: Octet3): ByteArray = x

    @GetMapping("/explicit-query/chain")
    fun explicitQueryChain(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/chain")
    fun varargQueryChain(@RequestParam vararg data: Octet3): String = ""

    @PostMapping("/explicit/chain-nullable")
    fun explicitChainNullable(@RequestBody x: Array<Byte?>): Array<Byte?> = x

    @PostMapping("/vararg/chain-nullable")
    fun varargChainNullable(@RequestBody vararg x: Top): Array<Byte?> = arrayOf(*x)

    @GetMapping("/explicit-query/chain-nullable")
    fun explicitQueryChainNullable(@RequestParam data: Array<Byte?>): String = ""

    @GetMapping("/vararg-query/chain-nullable")
    fun varargQueryChainNullable(@RequestParam vararg data: Top): String = ""

    @PostMapping("/explicit/qualified")
    fun explicitQualified(@RequestBody x: ByteArray): ByteArray = x

    @PostMapping("/vararg/qualified")
    fun varargQualified(@RequestBody vararg x: QualifiedByte): ByteArray = x

    @GetMapping("/explicit-query/qualified")
    fun explicitQueryQualified(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/qualified")
    fun varargQueryQualified(@RequestParam vararg data: QualifiedByte): String = ""

    @PostMapping("/explicit/generic")
    fun explicitGeneric(@RequestBody x: ByteArray): ByteArray = x

    @PostMapping("/vararg/generic")
    fun varargGeneric(@RequestBody vararg x: Const<String>): ByteArray = x

    @GetMapping("/explicit-query/generic")
    fun explicitQueryGeneric(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/generic")
    fun varargQueryGeneric(@RequestParam vararg data: Const<String>): String = ""

    @PostMapping("/explicit/generic-nullable")
    fun explicitGenericNullable(@RequestBody x: Array<Byte?>): Array<Byte?> = x

    @PostMapping("/vararg/generic-nullable")
    fun varargGenericNullable(@RequestBody vararg x: MaybeConst<String>): Array<Byte?> = arrayOf(*x)

    @GetMapping("/explicit-query/generic-nullable")
    fun explicitQueryGenericNullable(@RequestParam data: Array<Byte?>): String = ""

    @GetMapping("/vararg-query/generic-nullable")
    fun varargQueryGenericNullable(@RequestParam vararg data: MaybeConst<String>): String = ""

    @PostMapping("/explicit/private")
    fun explicitPrivate(@RequestBody x: ByteArray): ByteArray = x

    @PostMapping("/vararg/private")
    fun varargPrivate(@RequestBody vararg x: Secret): ByteArray = x

    @GetMapping("/explicit-query/private")
    fun explicitQueryPrivate(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/private")
    fun varargQueryPrivate(@RequestParam vararg data: Secret): String = ""

    @PostMapping("/explicit/plain-import")
    fun explicitPlainImport(@RequestBody x: ByteArray): ByteArray = x

    @PostMapping("/vararg/plain-import")
    fun varargPlainImport(@RequestBody vararg x: PlainOctet): ByteArray = x

    @GetMapping("/explicit-query/plain-import")
    fun explicitQueryPlainImport(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/plain-import")
    fun varargQueryPlainImport(@RequestParam vararg data: PlainOctet): String = ""

    @PostMapping("/explicit/renamed-import")
    fun explicitRenamedImport(@RequestBody x: ByteArray): ByteArray = x

    @PostMapping("/vararg/renamed-import")
    fun varargRenamedImport(@RequestBody vararg x: Aliased): ByteArray = x

    @GetMapping("/explicit-query/renamed-import")
    fun explicitQueryRenamedImport(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/renamed-import")
    fun varargQueryRenamedImport(@RequestParam vararg data: Aliased): String = ""

    @PostMapping("/explicit/star-import")
    fun explicitStarImport(@RequestBody x: ByteArray): ByteArray = x

    @PostMapping("/vararg/star-import")
    fun varargStarImport(@RequestBody vararg x: StarOctet): ByteArray = x

    @GetMapping("/explicit-query/star-import")
    fun explicitQueryStarImport(@RequestParam data: ByteArray): String = ""

    @GetMapping("/vararg-query/star-import")
    fun varargQueryStarImport(@RequestParam vararg data: StarOctet): String = ""

    @PostMapping("/explicit/star-letter")
    fun explicitStarLetter(@RequestBody x: CharArray): CharArray = x

    @PostMapping("/vararg/star-letter")
    fun varargStarLetter(@RequestBody vararg x: StarLetter): CharArray = x

    @GetMapping("/explicit-query/star-letter")
    fun explicitQueryStarLetter(@RequestParam data: CharArray): String = ""

    @GetMapping("/vararg-query/star-letter")
    fun varargQueryStarLetter(@RequestParam vararg data: StarLetter): String = ""

    @PostMapping("/explicit/other-nullable")
    fun explicitOtherNullable(@RequestBody x: Array<Byte?>): Array<Byte?> = x

    @PostMapping("/vararg/other-nullable")
    fun varargOtherNullable(@RequestBody vararg x: OtherMaybe): Array<Byte?> = arrayOf(*x)

    @GetMapping("/explicit-query/other-nullable")
    fun explicitQueryOtherNullable(@RequestParam data: Array<Byte?>): String = ""

    @GetMapping("/vararg-query/other-nullable")
    fun varargQueryOtherNullable(@RequestParam vararg data: OtherMaybe): String = ""

    @PostMapping("/explicit/text")
    fun explicitText(@RequestBody x: Array<String>): Array<String> = x

    @PostMapping("/vararg/text")
    fun varargText(@RequestBody vararg x: Text): Array<String> = arrayOf(*x)

    @GetMapping("/explicit-query/text")
    fun explicitQueryText(@RequestParam data: Array<String>): String = ""

    @GetMapping("/vararg-query/text")
    fun varargQueryText(@RequestParam vararg data: Text): String = ""

    @PostMapping("/explicit/blob")
    fun explicitBlob(@RequestBody x: Array<ByteArray>): Array<ByteArray> = x

    @PostMapping("/vararg/blob")
    fun varargBlob(@RequestBody vararg x: Blob): Array<ByteArray> = arrayOf(*x)

    @GetMapping("/explicit-query/blob")
    fun explicitQueryBlob(@RequestParam data: Array<ByteArray>): String = ""

    @GetMapping("/vararg-query/blob")
    fun varargQueryBlob(@RequestParam vararg data: Blob): String = ""

    @PostMapping("/explicit/shadowed")
    fun explicitShadowed(@RequestBody x: Array<ShadowByte>): Array<ShadowByte> = x

    @PostMapping("/vararg/shadowed")
    fun varargShadowed(@RequestBody vararg x: Shadowed): Array<ShadowByte> = arrayOf(*x)

    @GetMapping("/explicit-query/shadowed")
    fun explicitQueryShadowed(@RequestParam data: Array<ShadowByte>): String = ""

    @GetMapping("/vararg-query/shadowed")
    fun varargQueryShadowed(@RequestParam vararg data: Shadowed): String = ""

    // An alias that is not a vararg is the type it stands for, as it was.

    @PostMapping("/plain/octet")
    fun plainOctet(@RequestBody x: Octet): Octet = x

    @PostMapping("/plain/byte")
    fun plainByte(@RequestBody x: Byte): Byte = x

    @PostMapping("/plain/maybe-byte")
    fun plainMaybeByte(@RequestBody x: MaybeByte): MaybeByte = x

    @PostMapping("/plain/nullable-byte")
    fun plainNullableByte(@RequestBody x: Byte?): Byte? = x

    @PostMapping("/plain/letter")
    fun plainLetter(@RequestBody x: Letter): Letter = x

    @PostMapping("/plain/char")
    fun plainChar(@RequestBody x: Char): Char = x
}

package com.example.bytearray

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

// A property of the primary constructor declared `vararg val` holds the array of
// its elements: a ByteArray for `Byte`, a CharArray for `Char`, an Array of the
// element for `Byte?` and `String`. A class takes one vararg, so each has an
// explicit twin declared with the array Kotlin gives the property.

class VarargBytesForm(vararg val data: Byte)

class ExplicitBytesForm(val data: ByteArray)

class VarargCharsForm(vararg val data: Char)

class ExplicitCharsForm(val data: CharArray)

class VarargBoxedBytesForm(vararg val data: Byte?)

class ExplicitBoxedBytesForm(val data: Array<Byte?>)

class VarargBoxedCharsForm(vararg val data: Char?)

class ExplicitBoxedCharsForm(val data: Array<Char?>)

class VarargNamesForm(vararg val data: String)

class ExplicitNamesForm(val data: Array<String>)

class VarargIntsForm(vararg val data: Int)

class ExplicitIntsForm(val data: IntArray)

/** The element is an alias of a primitive type. */
class VarargOctetsForm(vararg val data: Octet)

class ExplicitOctetsForm(val data: ByteArray)

/** The element is an alias whose target is nullable. */
class VarargMaybeBytesForm(vararg val data: MaybeByte)

class ExplicitMaybeBytesForm(val data: Array<Byte?>)

/** A mutable property is one the data binder can set, and it holds the same array. */
class VarargMutableForm(vararg var data: Byte)

class ExplicitMutableForm(var data: ByteArray)

@RestController
@RequestMapping("/byte-vararg-forms")
class ByteArrayVarargFormController {

    @PostMapping("/explicit/bytes")
    fun explicitBytes(@RequestBody form: ExplicitBytesForm): ExplicitBytesForm = form

    @PostMapping("/vararg/bytes")
    fun varargBytes(@RequestBody form: VarargBytesForm): VarargBytesForm = form

    @PostMapping("/explicit/chars")
    fun explicitChars(@RequestBody form: ExplicitCharsForm): ExplicitCharsForm = form

    @PostMapping("/vararg/chars")
    fun varargChars(@RequestBody form: VarargCharsForm): VarargCharsForm = form

    @PostMapping("/explicit/boxed-bytes")
    fun explicitBoxedBytes(@RequestBody form: ExplicitBoxedBytesForm): ExplicitBoxedBytesForm = form

    @PostMapping("/vararg/boxed-bytes")
    fun varargBoxedBytes(@RequestBody form: VarargBoxedBytesForm): VarargBoxedBytesForm = form

    @PostMapping("/explicit/boxed-chars")
    fun explicitBoxedChars(@RequestBody form: ExplicitBoxedCharsForm): ExplicitBoxedCharsForm = form

    @PostMapping("/vararg/boxed-chars")
    fun varargBoxedChars(@RequestBody form: VarargBoxedCharsForm): VarargBoxedCharsForm = form

    @PostMapping("/explicit/names")
    fun explicitNames(@RequestBody form: ExplicitNamesForm): ExplicitNamesForm = form

    @PostMapping("/vararg/names")
    fun varargNames(@RequestBody form: VarargNamesForm): VarargNamesForm = form

    @PostMapping("/explicit/ints")
    fun explicitInts(@RequestBody form: ExplicitIntsForm): ExplicitIntsForm = form

    @PostMapping("/vararg/ints")
    fun varargInts(@RequestBody form: VarargIntsForm): VarargIntsForm = form

    @PostMapping("/explicit/octets")
    fun explicitOctets(@RequestBody form: ExplicitOctetsForm): ExplicitOctetsForm = form

    @PostMapping("/vararg/octets")
    fun varargOctets(@RequestBody form: VarargOctetsForm): VarargOctetsForm = form

    @PostMapping("/explicit/maybe-bytes")
    fun explicitMaybeBytes(@RequestBody form: ExplicitMaybeBytesForm): ExplicitMaybeBytesForm = form

    @PostMapping("/vararg/maybe-bytes")
    fun varargMaybeBytes(@RequestBody form: VarargMaybeBytesForm): VarargMaybeBytesForm = form

    @PostMapping("/explicit/mutable")
    fun explicitMutable(@RequestBody form: ExplicitMutableForm): ExplicitMutableForm = form

    @PostMapping("/vararg/mutable")
    fun varargMutable(@RequestBody form: VarargMutableForm): VarargMutableForm = form

    // The properties of a bound object are request parameters.

    @GetMapping("/explicit-bound/bytes")
    fun explicitBoundBytes(form: ExplicitBytesForm): String = ""

    @GetMapping("/vararg-bound/bytes")
    fun varargBoundBytes(form: VarargBytesForm): String = ""

    @GetMapping("/explicit-bound/chars")
    fun explicitBoundChars(form: ExplicitCharsForm): String = ""

    @GetMapping("/vararg-bound/chars")
    fun varargBoundChars(form: VarargCharsForm): String = ""

    @GetMapping("/explicit-bound/boxed-bytes")
    fun explicitBoundBoxedBytes(form: ExplicitBoxedBytesForm): String = ""

    @GetMapping("/vararg-bound/boxed-bytes")
    fun varargBoundBoxedBytes(form: VarargBoxedBytesForm): String = ""

    @GetMapping("/explicit-bound/names")
    fun explicitBoundNames(form: ExplicitNamesForm): String = ""

    @GetMapping("/vararg-bound/names")
    fun varargBoundNames(form: VarargNamesForm): String = ""

    @GetMapping("/explicit-bound/ints")
    fun explicitBoundInts(form: ExplicitIntsForm): String = ""

    @GetMapping("/vararg-bound/ints")
    fun varargBoundInts(form: VarargIntsForm): String = ""

    @GetMapping("/explicit-bound/octets")
    fun explicitBoundOctets(form: ExplicitOctetsForm): String = ""

    @GetMapping("/vararg-bound/octets")
    fun varargBoundOctets(form: VarargOctetsForm): String = ""

    @GetMapping("/explicit-bound/mutable")
    fun explicitBoundMutable(form: ExplicitMutableForm): String = ""

    @GetMapping("/vararg-bound/mutable")
    fun varargBoundMutable(form: VarargMutableForm): String = ""
}

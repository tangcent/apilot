package com.example.jdkscalar

import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

class JdkScalarForm(
    val primitiveShort: Short,
    val boxedShort: Short?,
    val qualifiedShort: java.lang.Short?,
    val shorts: ShortArray?,
    val boxedShorts: Array<Short>?,
    val qualifiedShorts: Array<java.lang.Short>?,
    val shortList: List<Short>?,
    val shortsByName: Map<String, Short>?,
    /** An unsigned short is an integer on the wire as well. */
    val unsignedShort: UShort?,
    val unsignedShorts: UShortArray?,
    /** A boxed type spelled with its package is the same type as the simple one. */
    val qualifiedInteger: java.lang.Integer?,
    val qualifiedLong: java.lang.Long?,
    val qualifiedFloat: java.lang.Float?,
    val qualifiedDouble: java.lang.Double?,
    val qualifiedBoolean: java.lang.Boolean?,
    /** Jackson writes any Number as a JSON number. */
    val number: Number?,
    val qualifiedNumber: java.lang.Number?,
)

@RestController
@RequestMapping("/jdk-scalars")
class JdkScalarController {

    @PostMapping("/save")
    fun save(@RequestBody form: JdkScalarForm): JdkScalarForm = form

    @PostMapping("/shorts")
    fun shorts(@RequestBody values: Array<Short>): ShortArray = ShortArray(0)

    @PostMapping("/count")
    fun count(@RequestBody value: Short): Short = value
}

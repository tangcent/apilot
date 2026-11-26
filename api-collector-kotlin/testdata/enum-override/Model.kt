package com.example.override

import com.fasterxml.jackson.annotation.JacksonAnnotationsInside
import com.fasterxml.jackson.annotation.JsonFormat
import com.fasterxml.jackson.annotation.JsonValue
import com.fasterxml.jackson.databind.annotation.JsonSerialize
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

interface Wire {
    @JsonValue
    fun code(): String
}

interface Base {
    @JsonValue
    fun code(): String
}

interface Mid : Base {
    @JsonValue(false)
    override fun code(): String
}

interface Two {
    @JsonValue
    fun code(): String

    @JsonValue
    fun wire(): String
}

interface Typed<T> {
    @JsonValue
    fun value(): T
}

interface TrueWire {
    @JsonValue
    fun code(): String
}

interface FalseWire {
    @JsonValue(false)
    fun code(): String
}

interface DefaultedWire {
    @JsonValue
    fun code(): String = "x"
}

@JsonFormat(shape = JsonFormat.Shape.NUMBER)
interface AsNum

@JsonFormat(shape = JsonFormat.Shape.OBJECT)
interface AsObj

@JsonFormat(shape = JsonFormat.Shape.ARRAY)
interface AsArr

@JsonFormat(shape = JsonFormat.Shape.STRING)
interface AsStr

@JsonSerialize(using = Lower::class)
interface AsSer

@JsonFormat(shape = JsonFormat.Shape.NUMBER)
interface NumBase

interface NumMid : NumBase

@JacksonAnnotationsInside
@JsonValue
annotation class AsValue

@JacksonAnnotationsInside
@JsonValue(false)
annotation class NotValue

@JacksonAnnotationsInside
@JsonFormat(shape = JsonFormat.Shape.NUMBER)
annotation class AsNumber

@Retention(AnnotationRetention.BINARY)
@JacksonAnnotationsInside
@JsonValue
annotation class AsValueClass

interface BundleWire {
    @AsValue
    fun code(): String
}

enum class Plain {
    A,
    B
}

enum class Off : Wire {
    A,
    B;

    @JsonValue(false)
    override fun code(): String = "x"
}

enum class Inherit : Wire {
    A,
    B;

    override fun code(): String = "x"
}

enum class ExplicitOn : Wire {
    A,
    B;

    @JsonValue(true)
    override fun code(): String = "x"
}

enum class MidOff : Mid {
    A,
    B;

    override fun code(): String = "x"
}

enum class Reopen : Mid {
    A,
    B;

    @JsonValue(true)
    override fun code(): String = "x"
}

enum class OwnOff {
    A,
    B;

    @JsonValue(false)
    fun code(): String = "x"
}

enum class OneOff : Two {
    A,
    B;

    @JsonValue(false)
    override fun code(): String = "x"

    override fun wire(): String = "y"
}

enum class Overload : Wire {
    A,
    B;

    @JsonValue(false)
    fun code(ignored: Int): String = "n"

    override fun code(): String = "x"
}

enum class GenericOn : Typed<String> {
    A,
    B;

    override fun value(): String = "x"
}

enum class GenericOff : Typed<String> {
    A,
    B;

    @JsonValue(false)
    override fun value(): String = "x"
}

enum class FirstTrue : TrueWire, FalseWire {
    A,
    B;

    override fun code(): String = "x"
}

enum class FirstFalse : FalseWire, TrueWire {
    A,
    B;

    override fun code(): String = "x"
}

enum class Defaulted : DefaultedWire {
    A,
    B
}

enum class PrivateVal {
    A,
    B;

    @JsonValue
    private fun code(): String = "x"
}

enum class IfaceNum : AsNum {
    A,
    B
}

enum class IfaceObj : AsObj {
    A,
    B
}

enum class IfaceArr : AsArr {
    A,
    B
}

enum class IfaceStr : AsStr {
    A,
    B
}

enum class IfaceSer : AsSer {
    A,
    B
}

@JsonFormat(shape = JsonFormat.Shape.STRING)
enum class StringOver : AsNum {
    A,
    B
}

@JsonSerialize(using = com.fasterxml.jackson.databind.JsonSerializer.None::class)
enum class NoneClears : AsSer {
    A,
    B
}

enum class ThroughNum : NumMid {
    A,
    B
}

enum class BundleVal {
    A,
    B;

    @AsValue
    fun code(): String = "x"
}

enum class BundleOff : Wire {
    A,
    B;

    @NotValue
    override fun code(): String = "x"
}

@AsNumber
enum class BundleNum {
    A,
    B
}

enum class BundleIface : BundleWire {
    A,
    B;

    override fun code(): String = "x"
}

enum class ClassRet {
    A,
    B;

    @AsValueClass
    fun code(): String = "x"
}

enum class ConstBody {
    A {
        @JsonValue
        fun extra(): String = "x"
    },
    B
}

class OverrideForm {
    var namePlain: Plain? = null
    var nameOff: Off? = null
    var nameMid: MidOff? = null
    var nameOwnOff: OwnOff? = null
    var nameGenericOff: GenericOff? = null
    var nameStringOver: StringOver? = null
    var nameNone: NoneClears? = null
    var nameBundleOff: BundleOff? = null
    var nameClassRet: ClassRet? = null
    var nameIfaceString: IfaceStr? = null
    var nameFirstFalse: FirstFalse? = null
    var limitConst: ConstBody? = null

    var valueInherit: Inherit? = null
    var valueExplicit: ExplicitOn? = null
    var valueReopen: Reopen? = null
    var valueOneOff: OneOff? = null
    var valueOverload: Overload? = null
    var valueGeneric: GenericOn? = null
    var valueNumber: IfaceNum? = null
    var valueObject: IfaceObj? = null
    var valueArray: IfaceArr? = null
    var valueSerialize: IfaceSer? = null
    var valueThrough: ThroughNum? = null
    var valueBundle: BundleVal? = null
    var valueBundleNum: BundleNum? = null
    var valueBundleIface: BundleIface? = null
    var valueDefault: Defaulted? = null
    var valuePrivate: PrivateVal? = null
    var valueFirstTrue: FirstTrue? = null
}

@RestController
@RequestMapping("/override")
class OverrideController {
    @PostMapping
    fun create(@RequestBody form: OverrideForm) {
    }

    @GetMapping
    fun page(query: OverrideForm) {
    }
}

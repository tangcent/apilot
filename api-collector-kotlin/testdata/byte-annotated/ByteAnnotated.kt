package com.example.byteannotated

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RequestPart
import org.springframework.web.bind.annotation.RequestHeader
import org.springframework.web.bind.annotation.CookieValue
import org.springframework.web.bind.annotation.RestController
import com.fasterxml.jackson.annotation.JsonFormat
import com.fasterxml.jackson.annotation.JsonRawValue
import com.fasterxml.jackson.databind.annotation.JsonSerialize

class ByteAnnotatedForm(
    val plain: ByteArray,
    val letters: CharArray,
    @JsonSerialize(using = Nums::class) val using: ByteArray,
    @get:JsonSerialize(using = Nums::class) val getter: ByteArray,
    @field:JsonSerialize(using = Nums::class) val field: ByteArray,
    @param:JsonSerialize(using = Nums::class) val param: ByteArray,
    @JsonSerialize(converter = Conv::class) val converter: ByteArray,
    @JsonSerialize(using = JsonSerializer.None::class) val none: ByteArray,
    @JsonSerialize(contentUsing = Nums::class) val contentSelf: ByteArray,
    @JsonSerialize(contentUsing = Nums::class) val chunks: Array<ByteArray>,
    @JsonSerialize(contentUsing = Nums::class) val blobs: List<ByteArray>,
    @JsonSerialize(contentUsing = Nums::class) val byName: Map<String, ByteArray>,
    @JsonRawValue val raw: ByteArray,
    @JsonFormat(shape = JsonFormat.Shape.ARRAY) val fmtArray: ByteArray,
    @JsonFormat(shape = JsonFormat.Shape.NUMBER) val fmtNumber: ByteArray,
    @JsonFormat(shape = JsonFormat.Shape.STRING) val fmtString: ByteArray,
    @JsonFormat(shape = JsonFormat.Shape.OBJECT) val fmtObject: ByteArray,
    @JsonSerialize(using = Nums::class) val lettersUsing: CharArray,
    val nullable: ByteArray?,
    val optional: java.util.Optional<ByteArray>,
)

@JsonFormat(shape = JsonFormat.Shape.ARRAY)
class ByteAnnotatedClassFmt(
    val data: ByteArray,
)

class ByteAnnotatedQuery(
    val data: ByteArray,
    val letters: CharArray,
)

open class ByteAnnotatedBase<T> {
    @PostMapping("/typed")
    open fun typed(@RequestBody vararg data: T): Array<out T> = data

    @PostMapping("/declared")
    open fun declared(@RequestBody vararg data: Byte): ByteArray = data
}

@RestController
@RequestMapping("/byte-annotated")
class ByteAnnotatedController {

    @PostMapping("/form")
    fun form(@RequestBody body: ByteAnnotatedForm): ByteAnnotatedForm = body

    @PostMapping("/class-format")
    fun classFormat(@RequestBody body: ByteAnnotatedClassFmt): ByteAnnotatedClassFmt = body

    @PostMapping("/whole")
    fun whole(@RequestBody body: ByteArray): ByteArray = body

    @PostMapping("/suspend")
    suspend fun suspended(@RequestBody body: ByteArray): ByteArray = body

    @GetMapping("/query")
    fun query(@RequestParam data: ByteArray): String = ""

    @GetMapping("/header")
    fun header(@RequestHeader data: ByteArray): String = ""

    @GetMapping("/cookie")
    fun cookie(@CookieValue data: ByteArray): String = ""

    @PostMapping("/part")
    fun part(@RequestPart data: ByteArray): String = ""

    @GetMapping("/bound")
    fun bound(query: ByteAnnotatedQuery): String = ""
}

@RestController
@RequestMapping("/byte-annotated/sub")
class ByteAnnotatedSub : ByteAnnotatedBase<Byte>()

@RestController
@RequestMapping("/byte-annotated/object")
object ByteAnnotatedObject {
    @PostMapping("/whole")
    fun whole(@RequestBody body: ByteArray): ByteArray = body
}

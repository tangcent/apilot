package com.example.platformtypes

import com.fasterxml.jackson.databind.JsonNode
import com.fasterxml.jackson.databind.node.ArrayNode
import com.fasterxml.jackson.databind.node.ObjectNode
import java.sql.Time
import java.sql.Timestamp
import java.time.DayOfWeek
import java.time.Month
import java.time.ZoneId
import java.util.Currency
import java.util.Locale
import java.util.Optional
import java.util.OptionalDouble
import java.util.OptionalInt
import java.util.OptionalLong
import java.util.TimeZone
import java.util.concurrent.atomic.AtomicBoolean
import java.util.concurrent.atomic.AtomicInteger
import java.util.concurrent.atomic.AtomicLong
import java.util.concurrent.atomic.LongAdder
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.multipart.MultipartFile

// The same form and handlers as
// api-collector-java/testdata/platform-types/.../PlatformTypesController.java.

class PlatformItem(
    val name: String,
    val qty: Int,
)

class PlatformTypesForm(
    // A collection or a map spelled with its package.
    val qualifiedList: kotlin.collections.List<String>,
    val qualifiedSet: java.util.Set<Long>,
    val qualifiedCollection: java.util.Collection<Int>,
    val qualifiedIterable: java.lang.Iterable<String>,
    val qualifiedMap: kotlin.collections.Map<String, Long>,
    val iterable: Iterable<String>,
    val linkedSet: java.util.LinkedHashSet<String>,
    val sortedMap: java.util.SortedMap<String, Int>,

    // A value that may be absent is written as the value, or left out.
    val optionalName: Optional<String>,
    val qualifiedOptional: java.util.Optional<Long>,
    val optionalItem: Optional<PlatformItem>,
    val optionalTags: Optional<List<String>>,
    val optionalInt: OptionalInt,
    val optionalLong: OptionalLong,
    val optionalDouble: OptionalDouble,

    // JDK values.
    val text: CharSequence,
    val locale: Locale,
    val currency: Currency,
    val timeZone: TimeZone,
    val zoneId: ZoneId,
    val day: DayOfWeek,
    val month: Month,
    val atomicInteger: AtomicInteger,
    val atomicLong: AtomicLong,
    val atomicBoolean: AtomicBoolean,
    val longAdder: LongAdder,
    val timestamp: Timestamp,
    val sqlDate: java.sql.Date,
    val sqlTime: Time,

    // Library classes.
    val jsonNode: JsonNode,
    val objectNode: ObjectNode,
    val arrayNode: ArrayNode,
    val file: MultipartFile,
)

class PlatformUploadForm {
    var title: String? = null
    var cover: MultipartFile? = null
    var attachments: List<MultipartFile>? = null
    var extras: Array<MultipartFile>? = null
}

@RestController
@RequestMapping("/platform-types")
class PlatformTypesController {

    @PostMapping("/save")
    fun save(@RequestBody form: PlatformTypesForm): PlatformTypesForm = form

    @PostMapping("/wildcard")
    fun wildcard(@RequestBody form: PlatformWildcardForm): PlatformWildcardForm = form

    @GetMapping("/search")
    fun search(keyword: Optional<String>, @RequestParam page: Optional<Int>,
               filter: Optional<PlatformItem>, currency: Currency, day: DayOfWeek): String = ""

    @PostMapping("/upload")
    fun upload(@RequestParam("file") file: MultipartFile, form: PlatformUploadForm,
               avatar: Optional<MultipartFile>): String = ""

    @PostMapping("/nodes")
    fun nodes(@RequestBody body: ObjectNode): JsonNode = body

    @PostMapping("/names")
    fun names(@RequestBody names: java.util.Set<String>): java.util.List<PlatformItem> = emptyList()

    @PostMapping("/maybe")
    fun maybe(@RequestBody item: Optional<PlatformItem>): Optional<PlatformItem> = item
}

package com.example.uploadidentity.own

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

/** A bean of the application that is called MultipartFile: bound property by property. */
class MultipartFile(
    val name: String?,
    val size: Long?,
)

/** Enums and beans of the application that are called like a Spring type. */
enum class HttpMethod { GET, POST }

enum class Locale { NORTH, SOUTH }

class Model(
    val label: String?,
)

class Sort(
    val by: String?,
)

class Optional<T>(
    val prefix: String?,
    val value: T?,
)

@RestController
@RequestMapping("/own")
class OwnController {

    @PostMapping("/single")
    fun single(file: MultipartFile): String = ""

    @PostMapping("/optional")
    fun optional(file: java.util.Optional<MultipartFile>): String = ""

    @PostMapping("/list")
    fun list(files: List<MultipartFile>): String = ""

    @GetMapping("/names")
    fun names(method: HttpMethod, model: Model, sort: Sort, locale: Locale): String = ""

    @GetMapping("/optional-param")
    fun optionalParam(@RequestParam("a") a: Optional<String>): String = ""

    @GetMapping("/own-optional")
    fun ownOptional(model: Optional<Model>): String = ""
}

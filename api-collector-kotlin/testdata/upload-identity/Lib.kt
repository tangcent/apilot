package com.example.uploadidentity.lib

import java.util.Optional
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RequestPart
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.multipart.MultipartFile

/** Spring's MultipartFile, by an import of the class, held by an Optional or not. */
@RestController
@RequestMapping("/lib")
class LibController {

    @PostMapping("/single")
    fun single(file: MultipartFile): String = ""

    @PostMapping("/optional")
    fun optional(avatar: Optional<MultipartFile>): String = ""

    @PostMapping("/list")
    fun list(files: List<MultipartFile>, docs: Array<MultipartFile>): String = ""

    @GetMapping("/optional-param")
    fun optionalParam(@RequestParam("a") a: Optional<String>): String = ""

    @PostMapping("/annotated")
    fun annotated(
        @RequestPart("cover") cover: Optional<MultipartFile>,
        @RequestParam("parts") parts: List<MultipartFile>,
    ): String = ""
}

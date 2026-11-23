package com.example.uploadidentity.demand

import java.util.*
import org.springframework.web.bind.annotation.*
import org.springframework.web.multipart.*

/** The class comes from an import on demand of its package. */
@RestController
@RequestMapping("/demand")
class DemandController {

    @PostMapping("/single")
    fun single(file: MultipartFile, avatar: Optional<MultipartFile>): String = ""

    @PostMapping("/qualified")
    fun qualified(file: org.springframework.web.multipart.MultipartFile): String = ""
}

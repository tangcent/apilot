package com.example.nesteddep

import com.example.nestlib.NestPage
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/nested-library")
class NestLibraryController {

    @GetMapping("/sort")
    fun sort(): NestPage.Sort = TODO()

    @GetMapping("/footer")
    fun footer(): com.example.nestlib.NestPage.Footer = TODO()

    @GetMapping("/page")
    fun page(): NestPage<NestRow> = TODO()

    @GetMapping("/missing")
    fun missing(): NestPage.Missing = TODO()
}

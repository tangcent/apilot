package com.example.nested.web

import com.example.nested.NestOther.*
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/nested-star")
class NestStarController {

    @GetMapping("/star")
    fun star(): Detail = TODO()

    @GetMapping("/fully-qualified")
    fun fullyQualified(): com.example.nested.NestHolder.Detail = TODO()
}

package com.example.nested

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/nested")
class NestController {

    @GetMapping("/holder")
    fun holder(): NestHolder = TODO()

    @GetMapping("/other")
    fun other(): NestOther = TODO()

    @GetMapping("/child")
    fun child(): NestChild = TODO()

    @GetMapping("/mid")
    fun mid(): NestHolder.Mid = TODO()

    @GetMapping("/detail")
    fun detail(): NestHolder.Detail = TODO()

    @PostMapping("/create")
    fun create(@RequestBody body: NestHolder.Detail): NestEnvelope<NestOther.Detail> = TODO()

    @GetMapping("/envelope")
    fun envelope(): NestEnvelope<NestHolder.Detail> = TODO()

    @GetMapping("/list")
    fun list(): List<NestOther.Detail> = TODO()
}

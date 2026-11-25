package com.example.mixscope

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/kotlin")
class KotlinController {

    @GetMapping("/dto")
    fun dto(): KotlinDto = TODO()

    @PostMapping("/save")
    fun save(@RequestBody dto: KotlinDto) {
    }
}

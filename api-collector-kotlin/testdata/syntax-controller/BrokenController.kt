package com.example.broken

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RestController

annotation class Marker(val groups: Array<String> = [])

@RestController
class BrokenController {
    @GetMapping("/broken")
    fun get(): String = "ok"
}

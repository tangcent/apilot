package com.example.broken

import org.springframework.web.bind.annotation.GetMapping

annotation class Marker(val groups: Array<String> = [])

@org.springframework.web.bind.annotation.RestController
class BrokenController {
    @GetMapping("/broken")
    fun get(): String = "ok"
}

package com.example.partial

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RestController

@RestController
class PartialController {
    @GetMapping("/first")
    fun first(): First = First()

    @GetMapping("/damaged")
    fun damaged(): Damaged = Damaged()

    @GetMapping("/third")
    fun third(): Third = Third()
}

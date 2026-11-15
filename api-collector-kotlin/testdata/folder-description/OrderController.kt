package com.example.folder

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/**
 * Order management.
 *
 * @author someone
 */
@RestController
@RequestMapping("/orders")
class OrderController {
    @GetMapping
    fun list(): String = ""
}

@RestController
class PingController {
    @GetMapping("/ping")
    fun ping(): String = ""
}

package com.example.deprecated

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/orders")
class OrderController {
    /**
     * Current lookup.
     */
    @GetMapping("/current")
    fun current(): String = ""

    /**
     * Old lookup.
     */
    @GetMapping("/old")
    @Deprecated("Use /orders/current", level = DeprecationLevel.WARNING)
    fun old(): String = ""
}

/**
 * Legacy endpoints.
 */
@RestController
@RequestMapping("/legacy")
@Deprecated(message = "Kept for old clients")
class LegacyController {
    /**
     * Ping.
     */
    @GetMapping("/ping")
    fun ping(): String = ""
}

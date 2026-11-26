package com.example.mixed

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

enum class KotlinKind {
    /** Sold to consumers. */
    RETAIL,
    WHOLESALE,
}

data class KotlinOrder(
    val orderNo: String,
    val amount: Long,
    val kind: KotlinKind,
)

class CreateOrderReq {
    var orderNo: String? = null
    var address: LegacyAddress? = null
    var status: LegacyStatus? = null
}

@RestController
@RequestMapping("/orders")
class OrderController {
    @PostMapping
    fun create(@RequestBody req: CreateOrderReq): KotlinOrder = KotlinOrder("1", 1, KotlinKind.RETAIL)

    @GetMapping("/address")
    fun address(): LegacyAddress = LegacyAddress()
}

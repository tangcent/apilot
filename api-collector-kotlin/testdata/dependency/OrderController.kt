package com.example.app

import com.example.lib.PageData
import com.example.lib.PageQuery
import com.example.lib.Wrapper
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RestController

enum class OrderStatus {
    OPEN,
    CLOSED
}

data class Order(
    val id: Long,
    val no: String?,
    val status: OrderStatus,
)

@RestController
class OrderController {
    @GetMapping("/orders")
    fun page(): Wrapper<PageData<Order>> = TODO()

    @GetMapping("/orders/search")
    fun search(query: PageQuery): Wrapper<PageData<Order>> = TODO()
}

package com.example.nested

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/nested")
class NestedController {

    @PostMapping
    fun create(@RequestBody holder: NestedHolder) {
    }

    @GetMapping
    fun page(query: NestedHolder) {
    }

    @GetMapping("/values")
    fun values(
        orderStatus: Order.Status,
        productStatus: Product.Status,
        productStatuses: Array<Product.Status>,
        orderStatusList: List<Order.Status>,
    ) {
    }
}

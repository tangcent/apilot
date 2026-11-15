package com.example.kdoc

import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

enum class Kind { PURCHASE, RECYCLE }

/**
 * Order creation request.
 */
data class CreateOrderReq(
    /**
     * Purchase order number.
     */
    val orderNo: String,
) {
    /**
     * Kind of the order.
     *
     * @see Kind
     */
    var kind: String? = null

    /** Single-line comment. */
    var remark: String? = null
}

@RestController
@RequestMapping("/kdoc")
class KdocController {
    /**
     * Create an order.
     */
    @PostMapping
    fun create(@RequestBody req: CreateOrderReq): CreateOrderReq = req
}

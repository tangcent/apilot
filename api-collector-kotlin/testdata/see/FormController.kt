package com.example.see

import io.swagger.annotations.ApiModelProperty
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/**
 * State of an order.
 */
enum class OrderState {
    /** Waiting for payment. */
    PENDING,
    /**
     * Paid.
     *
     * Ships within a day.
     */
    PAID,
    CLOSED
}

class OrderForm {
    /**
     * State name.
     *
     * @see OrderState
     */
    var state: String? = null

    /**
     * @see com.example.see.OrderState
     */
    var qualified: String? = null

    /** @see OrderState#PAID */
    var member: String? = null

    /** @see "the state table" */
    var quoted: String? = null

    /** @see OrderForm */
    var notAnEnum: String? = null

    /**
     * Values the field declares itself win.
     *
     * @see OrderState
     */
    @ApiModelProperty(allowableValues = "OPEN,SHUT")
    var declared: String? = null

    /** A property typed with the enum. */
    var typed: OrderState? = null
}

@RestController
@RequestMapping("/forms")
class FormController {

    @PostMapping
    fun create(@RequestBody form: OrderForm) {
    }

    @GetMapping
    fun page(query: OrderForm) {
    }
}

package com.example.required

import jakarta.validation.constraints.NotBlank
import jakarta.validation.constraints.NotNull
import org.jetbrains.annotations.Nullable
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

class CreateOrderReq {
    @NotBlank
    var orderNo: String? = null

    @NotNull
    var amount: Long? = null

    var remark: String? = null

    @Nullable
    var coupon: String? = null
}

data class OrderResp(
    val orderNo: String,
    @NotNull val status: String,
)

@RestController
@RequestMapping("/orders")
class OrderController {
    @PostMapping
    fun create(@RequestBody req: CreateOrderReq): OrderResp = OrderResp("1", "ok")
}

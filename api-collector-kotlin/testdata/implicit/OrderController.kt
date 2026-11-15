package com.example.implicit

import jakarta.servlet.http.HttpServletRequest
import jakarta.validation.Valid
import jakarta.validation.constraints.NotNull
import org.springframework.data.domain.Pageable
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.multipart.MultipartFile

enum class Status { ACTIVE, CLOSED }

class Address {
    var city: String? = null
}

class OrderQuery {
    /**
     * Order number.
     */
    var orderNo: String? = null
    var regionList: List<String>? = null
    var address: Address? = null
}

class Operator {
    var id: Long? = null
}

@RestController
@RequestMapping("/orders")
class OrderController {

    @GetMapping
    fun page(@Valid query: OrderQuery, pageable: Pageable, request: HttpServletRequest,
             @NotNull keyword: String?, ids: Array<Long>?, status: Status?): String = ""

    @PostMapping("/form")
    fun submit(form: OrderQuery, attachment: MultipartFile): String = ""

    @PostMapping("/json")
    fun json(@RequestBody body: OrderQuery, trace: String?): String = ""

    @GetMapping("/me")
    fun me(@CurrentUser operator: Operator, id: Long?): String = ""
}

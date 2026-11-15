package com.example.implicit

import com.fasterxml.jackson.annotation.JsonIgnore
import com.fasterxml.jackson.annotation.JsonProperty
import io.swagger.v3.oas.annotations.media.Schema
import jakarta.servlet.http.HttpServletRequest
import jakarta.validation.Valid
import jakarta.validation.constraints.NotNull
import java.time.LocalDate
import org.springframework.data.domain.Pageable
import org.springframework.data.domain.Sort
import org.springframework.format.annotation.DateTimeFormat
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.ModelAttribute
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController
import org.springframework.web.multipart.MultipartFile

// The same handlers as api-collector-java/testdata/implicit/.../OrderController.java.

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

    @JsonProperty("display_name")
    var displayName: String? = null

    @JsonIgnore
    var secret: String? = null

    /** Read-only: Spring cannot bind it. */
    val summary: String
        get() = orderNo ?: ""
}

class OrderFilter {
    var name: String? = null
}

class Operator {
    var id: Long? = null
}

@RestController
@RequestMapping("/orders")
class OrderController {

    @GetMapping
    fun page(@Valid query: OrderQuery, pageable: Pageable, request: HttpServletRequest,
             @NotNull keyword: String?, ids: List<Long>?,
             @Schema(defaultValue = "ACTIVE") status: Status?,
             @DateTimeFormat(iso = DateTimeFormat.ISO.DATE) from: LocalDate?): String = ""

    @PostMapping("/form")
    fun submit(form: OrderQuery, attachments: List<MultipartFile>): String = ""

    @PostMapping("/json")
    fun json(@RequestBody body: OrderQuery, trace: String?): String = ""

    @GetMapping("/me")
    fun me(@CurrentUser operator: Operator, id: Long?): String = ""

    @GetMapping("/search")
    fun search(@Audit @ModelAttribute filter: OrderFilter, sort: Sort): String = ""
}

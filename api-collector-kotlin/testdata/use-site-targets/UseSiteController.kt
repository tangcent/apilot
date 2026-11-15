package com.example.usesite

import com.fasterxml.jackson.annotation.JsonIgnore
import com.fasterxml.jackson.annotation.JsonInclude
import com.fasterxml.jackson.annotation.JsonProperty
import jakarta.validation.Valid
import jakarta.validation.constraints.NotBlank
import jakarta.validation.constraints.NotNull
import jakarta.validation.constraints.Size
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

class Item {
    @field:NotBlank
    var sku: String? = null
}

class CreateReq {
    /**
     * Display name.
     */
    @field:NotNull
    @field:Size(max = 64)
    var name: String? = null

    @get:JsonProperty("display_code")
    var code: String? = null

    @field:JsonProperty("tag") @get:JsonProperty("tag")
    var label: String? = null

    @get:JsonIgnore
    var secret: String? = null

    @get:JsonInclude(JsonInclude.Include.NON_NULL)
    @set:JvmName("assignNote")
    var note: String? = null

    @field:Valid
    var items: List<Item>? = null
}

data class CreateResp(
    @param:JsonProperty("id") @get:JsonProperty("id") val identifier: Long,
    @field:NotNull val status: String,
)

@RestController
@RequestMapping("/use-site")
class UseSiteController {
    @PostMapping
    fun create(@RequestBody req: CreateReq): CreateResp = CreateResp(1, "ok")
}

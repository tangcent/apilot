package com.example.invalid

import com.fasterxml.jackson.annotation.JsonProperty
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

data class InvalidReq(
    @get:JsonProperty("renamed") val original: String,
)

@RestController
@RequestMapping("/invalid")
class InvalidController {
    @PostMapping
    fun create(@RequestBody req: InvalidReq): InvalidReq = req
}

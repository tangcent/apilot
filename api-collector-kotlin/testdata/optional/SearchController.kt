package com.example.optional

import javax.validation.constraints.NotNull
import org.springframework.lang.Nullable
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestHeader
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/search")
class SearchController {

    @GetMapping
    fun search(
        @RequestParam keyword: String,
        @RequestParam tag: String?,
        @RequestParam size: Int = 20,
        @RequestParam @Nullable cursor: String,
        @RequestParam(required = true) region: String?,
        @RequestParam @NotNull ownerId: Long?,
        @RequestHeader("X-Trace") trace: String?,
        @RequestHeader("X-Tenant") tenant: String
    ) {
    }
}

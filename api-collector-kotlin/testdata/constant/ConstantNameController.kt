package com.example.constant

import com.example.constant.shared.Headers
import com.example.constant.shared.Params
import com.example.constant.shared.TENANT
import org.springframework.web.bind.annotation.CookieValue
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable
import org.springframework.web.bind.annotation.RequestHeader
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/constant-names")
class ConstantNameController {

    @GetMapping("/{orderNo}")
    fun find(
        @RequestHeader(TRACE) trace: String,
        @RequestHeader(Headers.VERSION) version: String,
        @RequestHeader(value = Headers.LANGUAGE, defaultValue = "en") language: String,
        @RequestParam(Params.PAGE_SIZE) size: Int,
        @PathVariable(name = Params.ORDER_NO) no: String,
        @CookieValue(Params.SESSION) session: String,
        @RequestParam(com.example.constant.shared.Params.KEYWORD) keyword: String,
        @RequestParam(TENANT) tenant: String,
        @RequestParam(REGION) region: String,
        @RequestHeader(Missing.NAME) fallback: String,
        @RequestParam("literal") literal: String
    ) {
    }

    companion object {
        const val TRACE = "X-Trace-Id"
    }
}

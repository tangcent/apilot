package com.example.nested.web

import com.example.nested.NestHolder.Detail
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/nested-import")
class NestImportController {

    @GetMapping("/single")
    fun single(): Detail = TODO()
}

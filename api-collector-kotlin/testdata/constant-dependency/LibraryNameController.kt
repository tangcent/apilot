package com.example.constantdep

import com.example.lib.CompiledHeaders
import com.example.lib.LibHeaders
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestHeader
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/library-names")
class LibraryNameController {

    @GetMapping
    fun list(
        @RequestHeader(LibHeaders.CHANNEL) channel: String,
        @RequestHeader(CompiledHeaders.DEVICE) device: String
    ) {
    }
}

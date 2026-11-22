package com.example.optionalscope.web

import com.example.optionalscope.own.Optional
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

// Optional is the class imported by name.
@RestController
@RequestMapping("/optional-scope")
class ImportedController {

    @GetMapping("/imported")
    fun imported(filter: Optional<String>): String = ""
}

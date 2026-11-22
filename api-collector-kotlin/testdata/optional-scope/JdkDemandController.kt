package com.example.optionalscope.jdk

import java.util.*
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

// Optional is java.util.Optional, imported on demand: the project's own Optional
// is in another package that this file does not import.
@RestController
@RequestMapping("/optional-scope")
class JdkDemandController {

    @GetMapping("/jdk-demand")
    fun demand(filter: Optional<String>): String = ""
}

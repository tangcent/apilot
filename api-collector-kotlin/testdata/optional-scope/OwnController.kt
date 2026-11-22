package com.example.optionalscope.own

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

// The same declarations as api-collector-java/testdata/optional-scope.

// A class of the project that has the simple name of java.util.Optional.
class Optional<T>(
    val prefix: String,
    val value: T,
)

// Optional is the class of this package.
@RestController
@RequestMapping("/optional-scope")
class OwnController {

    @GetMapping("/own")
    fun own(filter: Optional<String>): String = ""
}

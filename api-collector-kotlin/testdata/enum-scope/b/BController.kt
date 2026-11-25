package com.example.scope.b

import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

class Holder {
    var code: Code? = null

    var codes: List<Code>? = null

    var statuses: List<Order.Status>? = null
}

@RestController
@RequestMapping("/scope/b")
class BController {

    @PostMapping
    fun createB(@RequestBody holder: Holder) {
    }
}

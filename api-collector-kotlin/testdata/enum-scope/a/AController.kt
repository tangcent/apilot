package com.example.scope.a

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
@RequestMapping("/scope/a")
class AController {

    @PostMapping
    fun createA(@RequestBody holder: Holder) {
    }
}

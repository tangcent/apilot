package com.example.recovery

import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

class SortReq {
    var group: SortGroup? = null
}

@RestController
@RequestMapping("/recovery")
class RecoveryController {
    @PostMapping("/settlements")
    fun query(@RequestBody query: SettlementQuery): SettlementQuery = query

    @PostMapping("/sort")
    fun sort(@RequestBody req: SortReq): SortReq = req
}

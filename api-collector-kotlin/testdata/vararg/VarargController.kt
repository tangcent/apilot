package com.example.declarator

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

class DeclaratorItem(
    val name: String?,
)

@RestController
@RequestMapping("/declarators")
class VarargController {

    @GetMapping("/search")
    fun search(@RequestParam(required = false) vararg ids: String): Array<String> = arrayOf(*ids)

    @GetMapping("/search-array")
    fun searchArray(@RequestParam(required = false) ids: Array<String>?): Array<String> = emptyArray()

    @GetMapping("/implicit")
    fun implicit(vararg tags: String): Array<String> = arrayOf(*tags)

    @PostMapping("/items")
    fun items(@RequestBody vararg items: DeclaratorItem): Array<DeclaratorItem> = arrayOf(*items)

    @PostMapping("/items-array")
    fun itemsArray(@RequestBody items: Array<DeclaratorItem>): Array<DeclaratorItem> = items
}

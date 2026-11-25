package com.example.typearg.app

import com.example.typearg.dto.Item
import com.example.typearg.lib.Box
import com.example.typearg.lib.Holder
import com.example.typearg.lib.Outer
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/type-args")
class TypeArgumentController {

    @PostMapping("/box")
    fun box(@RequestBody body: Box<Item>): Box<Item> = body

    @GetMapping("/outer")
    fun outer(): Outer<Item> = TODO()

    @GetMapping("/outer-int")
    fun outerInt(): Outer<Int> = TODO()

    @GetMapping("/holder")
    fun holder(): Holder = TODO()
}

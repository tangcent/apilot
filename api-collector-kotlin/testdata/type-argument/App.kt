package com.example.typearg.app

import com.example.typearg.dto.Item
import com.example.typearg.lib.BaseController
import com.example.typearg.lib.Box
import com.example.typearg.lib.Holder
import com.example.typearg.lib.MidController
import com.example.typearg.lib.Outer
import java.time.Period
import java.util.Date
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

    @GetMapping("/box-list")
    fun boxList(): Box<List<Item>> = TODO()

    @GetMapping("/box-array")
    fun boxArray(): Box<Array<Item>> = TODO()

    @GetMapping("/box-map")
    fun boxMap(): Box<Map<String, Item>> = TODO()

    @GetMapping("/box-period")
    fun boxPeriod(): Box<Period> = TODO()

    @GetMapping("/box-date")
    fun boxDate(): Box<Date> = TODO()

    @GetMapping("/box-date-list")
    fun boxDateList(): Box<List<Date>> = TODO()

    @GetMapping("/box-string")
    fun boxString(): Box<String> = TODO()

    @GetMapping("/holder")
    fun holder(): Holder = TODO()
}

@RestController
@RequestMapping("/items")
class ItemController : BaseController<Item>()

@RestController
@RequestMapping("/lists")
class ItemListController : MidController<Item>()

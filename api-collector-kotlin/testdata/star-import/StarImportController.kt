package com.example.ondemand

import com.example.lib.model.*
import com.example.lib.response.*
import org.springframework.web.bind.annotation.*

@RestController
@RequestMapping("/star-import")
class StarImportController {

    @PostMapping("/create")
    fun create(@RequestBody order: StarOrder): StarEnvelope<List<StarItem>> = TODO()

    @GetMapping("/missing")
    fun missing(): StarEnvelope<StarMissing> = TODO()
}

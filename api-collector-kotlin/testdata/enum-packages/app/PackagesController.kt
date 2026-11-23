package com.example.pkgs.app

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

class PackagesForm {
    var own: Status? = null

    var ownLib: com.example.pkgs.lib.Status? = null

    var many: List<Status>? = null
}

@RestController
@RequestMapping("/pkgs")
class PackagesController {

    @PostMapping
    fun create(@RequestBody form: PackagesForm) {
    }

    @GetMapping("/direct")
    fun direct(): Status = TODO()

    @GetMapping("/list")
    fun list(): List<Status> = TODO()
}

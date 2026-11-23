package com.example.kcunit

import org.springframework.web.bind.annotation.DeleteMapping
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RequestParam
import org.springframework.web.bind.annotation.RestController

@RestController
@RequestMapping("/unit")
class Controller {

    /** Written without a return type: the function returns the standard Unit. */
    @DeleteMapping("/block")
    fun block(@RequestParam id: Long) {
    }

    @DeleteMapping("/qualified")
    fun qualified(@RequestParam id: Long): kotlin.Unit {
    }

    /** Declared as Unit, which is the enum of this package. */
    @GetMapping("/enum")
    fun enum(): Unit = TODO()

    @PostMapping("/declared")
    fun declared(@RequestParam id: Long): Unit {
    }
}

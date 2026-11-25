package com.example.kcnest

import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/** Classes nested here are in scope in this class's body only. */
@RestController
@RequestMapping("/holder")
class Holder {
    class Int(val own: kotlin.String)

    enum class Unit { A, B }

    /** Int is the class nested above. */
    @GetMapping("/count")
    fun count(): Int = TODO()

    @GetMapping("/unit")
    fun unit(): Unit = TODO()
}

/** The nested classes of a class are in scope in its subclasses too. */
open class Base {
    class Number(val own: kotlin.String)
}

@RestController
@RequestMapping("/child")
class Child : Base() {
    @GetMapping("/weight")
    fun weight(): Number = TODO()
}

class Other {
    class Number(val own: kotlin.String)
}

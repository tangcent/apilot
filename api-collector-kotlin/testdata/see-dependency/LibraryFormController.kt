package com.example.seedep

import com.example.lib.OrderState
import com.example.lib.PayChannel
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

class LibraryOrderForm {
    /**
     * State name.
     *
     * @see OrderState
     */
    var state: String? = null

    /**
     * @see com.example.lib.OrderState
     */
    var qualified: String? = null

    /** @see PayChannel */
    var channel: String? = null

    /** @see com.example.lib.OrderState#PAID */
    var member: String? = null

    /** @see com.example.lib.PageQuery */
    var notAnEnum: String? = null

    /** @see com.example.lib.Missing */
    var missing: String? = null

    /** @see UnimportedState */
    var unimported: String? = null

    /** @see java.time.DayOfWeek */
    var platform: String? = null

    /** A property typed with the library's enum. */
    var typed: OrderState? = null
}

@RestController
@RequestMapping("/library-forms")
class LibraryFormController {

    @PostMapping
    fun create(@RequestBody form: LibraryOrderForm) {
    }

    @GetMapping
    fun page(query: LibraryOrderForm) {
    }
}

package com.example.enumdep

import com.example.lib.Currency
import com.example.lib.Missing
import com.example.lib.Mode
import com.example.lib.OrderState
import com.example.lib.PayChannel
import com.example.lib.Region
import com.example.lib.Stage
import com.example.lib.Tier
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

class EnumLibraryForm {
    /** A property typed with an enum of the library. */
    var state: OrderState? = null

    var states: List<OrderState>? = null

    /** An enum of the library that was decompiled. */
    var channel: PayChannel? = null

    /** An enum of the library that was decompiled from Kotlin. */
    var region: Region? = null

    /** An enum of the library that declares @JsonValue. */
    var currency: Currency? = null

    /** An enum of the library that was decompiled from Kotlin and declares @JsonValue. */
    var tier: Tier? = null

    /** An enum of the library that was decompiled and gives a constant a name with @JsonProperty. */
    var stage: Stage? = null

    /** An enum of the library that was decompiled from Kotlin and gives a constant a name with @JsonProperty. */
    var mode: Mode? = null

    /** A class the library does not have. */
    var missing: Missing? = null
}

@RestController
@RequestMapping("/enum-library")
class EnumLibraryController {

    @PostMapping
    fun create(@RequestBody form: EnumLibraryForm) {
    }
}

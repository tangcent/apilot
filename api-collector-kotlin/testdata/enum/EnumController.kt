package com.example.enumjson

import com.example.lib.Missing
import com.fasterxml.jackson.annotation.JsonFormat
import com.fasterxml.jackson.annotation.JsonValue
import io.swagger.annotations.ApiModelProperty
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/**
 * State of an order.
 */
enum class OrderState {
    /** Waiting for payment. */
    PENDING,
    /**
     * Paid.
     *
     * Ships within a day.
     */
    PAID,
    CLOSED
}

/** Carries a code, but Jackson writes the name of the constant. */
enum class Level(val code: String) {
    LOW("l"),
    HIGH("h"),
}

/** Jackson writes what the annotated getter returns. */
enum class ChannelCode(@get:JsonValue val code: String) {
    ALIPAY("alipay"),
    WECHAT("wechat"),
}

/** Jackson writes what the annotated field holds. */
enum class Grade(@JsonValue val label: String) {
    A("a"),
    B("b"),
}

/** Jackson writes an object. */
@JsonFormat(shape = JsonFormat.Shape.OBJECT)
enum class Shape {
    CIRCLE,
    SQUARE,
}

/** Jackson writes the ordinal. */
@JsonFormat(shape = JsonFormat.Shape.NUMBER)
enum class Rank {
    FIRST,
    SECOND,
}

/** The string shape is what Jackson writes anyway. */
@JsonFormat(shape = JsonFormat.Shape.STRING)
enum class Flag {
    ON,
    OFF,
}

/** The annotation is switched off. */
enum class Toggle(val code: String) {
    YES("y"),
    NO("n"),
    ;

    @JsonValue(false)
    fun toCode(): String = code
}

class EnumForm {
    /** State of the order. */
    var state: OrderState? = null

    var history: List<OrderState>? = null

    var allowed: Set<OrderState>? = null

    var byRegion: Map<String, OrderState>? = null

    var byState: Map<OrderState, String>? = null

    var level: Level? = null

    var channel: ChannelCode? = null

    var channels: List<ChannelCode>? = null

    var grade: Grade? = null

    var shape: Shape? = null

    var rank: Rank? = null

    var flag: Flag? = null

    var toggle: Toggle? = null

    var missing: Missing? = null

    /** Values the field declares itself win. */
    @ApiModelProperty(allowableValues = "OPEN,SHUT")
    var declared: OrderState? = null
}

class EnumEnvelope<T> {
    /** Business code. */
    var code: Int = 0

    var data: T? = null
}

@RestController
@RequestMapping("/enums")
class EnumController {

    @PostMapping
    fun create(@RequestBody form: EnumForm): EnumEnvelope<OrderState> {
        return EnumEnvelope()
    }

    @GetMapping
    fun page(query: EnumForm) {
    }

    @GetMapping("/state")
    fun state(): OrderState {
        return OrderState.PENDING
    }

    @GetMapping("/states")
    fun states(): List<OrderState> {
        return emptyList()
    }

    @GetMapping("/channel")
    fun channel(): ChannelCode {
        return ChannelCode.ALIPAY
    }
}

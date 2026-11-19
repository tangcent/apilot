package com.example.enumjson;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonValue;

/**
 * State of an order.
 */
enum OrderState {
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
enum Level {
    LOW("l"),
    HIGH("h");

    private final String code;

    Level(String code) {
        this.code = code;
    }

    public String getCode() {
        return code;
    }
}

/** Jackson writes what the annotated method returns. */
enum ChannelCode {
    ALIPAY("alipay"),
    WECHAT("wechat");

    private final String code;

    ChannelCode(String code) {
        this.code = code;
    }

    @JsonValue
    public String getCode() {
        return code;
    }
}

/** Jackson writes what the annotated field holds. */
enum Grade {
    A("a"),
    B("b");

    @JsonValue
    private final String label;

    Grade(String label) {
        this.label = label;
    }
}

/** Jackson writes an object. */
@JsonFormat(shape = JsonFormat.Shape.OBJECT)
enum Shape {
    CIRCLE,
    SQUARE
}

/** Jackson writes the ordinal. */
@JsonFormat(shape = JsonFormat.Shape.NUMBER)
enum Rank {
    FIRST,
    SECOND
}

/** The string shape is what Jackson writes anyway. */
@JsonFormat(shape = JsonFormat.Shape.STRING)
enum Flag {
    ON,
    OFF
}

/** The annotation is switched off. */
enum Toggle {
    YES("y"),
    NO("n");

    private final String code;

    Toggle(String code) {
        this.code = code;
    }

    @JsonValue(false)
    public String getCode() {
        return code;
    }
}

package com.example.see;

import io.swagger.annotations.ApiModelProperty;

public class OrderForm {
    /**
     * State name.
     *
     * @see OrderState
     */
    private String state;

    /**
     * @see com.example.see.OrderState
     */
    private String qualified;

    /** @see OrderState#PAID */
    private String member;

    /** @see "the state table" */
    private String quoted;

    /** @see OrderForm */
    private String notAnEnum;

    /**
     * Values the field declares itself win.
     *
     * @see OrderState
     */
    @ApiModelProperty(allowableValues = "OPEN,SHUT")
    private String declared;
}

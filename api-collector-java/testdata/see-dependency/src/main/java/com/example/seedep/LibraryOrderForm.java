package com.example.seedep;

import com.example.lib.OrderState;
import com.example.lib.PayChannel;

public class LibraryOrderForm {
    /**
     * State name.
     *
     * @see OrderState
     */
    private String state;

    /**
     * @see com.example.lib.OrderState
     */
    private String qualified;

    /** @see PayChannel */
    private String channel;

    /** @see com.example.lib.OrderState#PAID */
    private String member;

    /** @see com.example.lib.PageQuery */
    private String notAnEnum;

    /** @see com.example.lib.Missing */
    private String missing;

    /** @see UnimportedState */
    private String unimported;

    /** @see java.time.DayOfWeek */
    private String platform;

    /** A field typed with the library's enum. */
    private OrderState typed;
}

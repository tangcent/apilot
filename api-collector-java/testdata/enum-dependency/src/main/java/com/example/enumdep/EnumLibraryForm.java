package com.example.enumdep;

import com.example.lib.Currency;
import com.example.lib.Missing;
import com.example.lib.OrderState;
import com.example.lib.PayChannel;
import com.example.lib.Region;
import com.example.lib.Tier;
import java.util.List;

public class EnumLibraryForm {
    /** A field typed with an enum of the library. */
    private OrderState state;

    private List<OrderState> states;

    /** An enum of the library that was decompiled. */
    private PayChannel channel;

    /** An enum of the library that was decompiled from Kotlin. */
    private Region region;

    /** An enum of the library that declares @JsonValue. */
    private Currency currency;

    /** An enum of the library that was decompiled from Kotlin and declares @JsonValue. */
    private Tier tier;

    /** A class the library does not have. */
    private Missing missing;
}

package com.example.enumdep;

import com.example.lib.Currency;
import com.example.lib.Missing;
import com.example.lib.Mode;
import com.example.lib.OrderState;
import com.example.lib.PayChannel;
import com.example.lib.Region;
import com.example.lib.Stage;
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

    /** An enum of the library that was decompiled and gives a constant a name with @JsonProperty. */
    private Stage stage;

    /** An enum of the library that was decompiled from Kotlin and gives a constant a name with @JsonProperty. */
    private Mode mode;

    /** A class the library does not have. */
    private Missing missing;
}

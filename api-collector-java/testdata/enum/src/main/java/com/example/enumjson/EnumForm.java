package com.example.enumjson;

import com.example.lib.Missing;
import io.swagger.annotations.ApiModelProperty;
import java.util.List;
import java.util.Map;
import java.util.Set;

public class EnumForm {
    /** State of the order. */
    private OrderState state;

    private List<OrderState> history;

    private Set<OrderState> allowed;

    private Map<String, OrderState> byRegion;

    private Map<OrderState, String> byState;

    private Level level;

    private ChannelCode channel;

    private List<ChannelCode> channels;

    private Grade grade;

    private Shape shape;

    private Rank rank;

    private Flag flag;

    private Toggle toggle;

    private Missing missing;

    /** Values the field declares itself win. */
    @ApiModelProperty(allowableValues = "OPEN,SHUT")
    private OrderState declared;
}

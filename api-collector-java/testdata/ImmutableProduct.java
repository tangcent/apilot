package com.example.demo.model;

import java.math.BigDecimal;

/**
 * An immutable DTO in the style Lombok's {@code @Value} generates: every
 * instance field is final, so the resolver must not treat final as constant.
 */
public class ImmutableProduct {

    public static final String TYPE = "product";

    private final String sku;
    private final BigDecimal price;

    public ImmutableProduct(String sku, BigDecimal price) {
        this.sku = sku;
        this.price = price;
    }

    public String getSku() {
        return sku;
    }

    public BigDecimal getPrice() {
        return price;
    }
}

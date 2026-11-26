package com.example.nested;

import java.util.List;
import java.util.Map;

public class NestedHolder {
    private Order.Status order;

    private List<Order.Status> orders;

    private Order.Status[] orderArray;

    private Product.Status product;

    private List<Product.Status> products;

    private Product.Status[] productArray;

    private Map<String, Product.Status> byName;
}

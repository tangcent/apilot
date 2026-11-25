package com.example.nested

import com.fasterxml.jackson.annotation.JsonValue

/** Two classes of one file, each nesting an enum named Status that declares the constant A. */
class Order {

    /** Jackson writes what the annotated function returns. */
    enum class Status {
        A,
        B;

        @JsonValue
        fun text(): String = name.lowercase()
    }
}

class Product {

    /** Jackson writes the name of the entry. */
    enum class Status {
        A,
        C
    }
}

class NestedHolder {
    var order: Order.Status? = null

    var orders: List<Order.Status>? = null

    var orderArray: Array<Order.Status>? = null

    var product: Product.Status? = null

    var products: List<Product.Status>? = null

    var productArray: Array<Product.Status>? = null

    var byName: Map<String, Product.Status>? = null
}

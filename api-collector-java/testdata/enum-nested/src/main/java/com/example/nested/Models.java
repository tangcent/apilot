package com.example.nested;

import com.fasterxml.jackson.annotation.JsonValue;

/** Two classes of one file, each nesting an enum named Status that declares the constant A. */
class Order {

    /** Jackson writes what the annotated method returns. */
    enum Status {
        A,
        B;

        @JsonValue
        String text() {
            return name().toLowerCase();
        }
    }
}

class Product {

    /** Jackson writes the name of the constant. */
    enum Status {
        A,
        C
    }
}

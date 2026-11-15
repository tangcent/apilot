package com.example.wildcard;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/orders")
public class OrderController {

    /**
     * Page of order lines.
     */
    @GetMapping("/lines")
    public PageData<OrderLine> lines() {
        return null;
    }

    /**
     * Lines held as wildcard collections.
     */
    @GetMapping("/bounds")
    public LineBounds bounds() {
        return null;
    }

    public static class PageData<T> {
        /** Total number of rows. */
        private long total;
        /** Rows of this page; every element is a T. */
        private List<? extends T> rows;
    }

    public static class LineBounds {
        /** Every element is an OrderLine. */
        private List<? extends OrderLine> covariant;
        /** Any supertype of OrderLine: the element type is open. */
        private List<? super OrderLine> contravariant;
        /** Unknown element type. */
        private List<?> open;
    }

    public static class OrderLine {
        /** Stock keeping unit. */
        private String sku;
        /** Ordered quantity. */
        private int quantity;
    }
}

package com.example.deprecated;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/orders")
public class OrderController {

    /**
     * Current lookup.
     */
    @GetMapping("/current")
    public String current() {
        return "";
    }

    /**
     * Old lookup.
     */
    @Deprecated
    @GetMapping("/old")
    public String old() {
        return "";
    }

    /**
     * Legacy lookup.
     *
     * @deprecated use /orders/current instead
     */
    @GetMapping("/legacy")
    public String legacy() {
        return "";
    }
}

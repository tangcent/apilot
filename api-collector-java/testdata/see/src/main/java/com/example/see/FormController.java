package com.example.see;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/forms")
public class FormController {

    @PostMapping
    public void create(@RequestBody OrderForm form) {
    }

    @GetMapping
    public void page(OrderForm query) {
    }
}

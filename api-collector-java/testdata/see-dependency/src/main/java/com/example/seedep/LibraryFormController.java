package com.example.seedep;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/library-forms")
public class LibraryFormController {

    @PostMapping
    public void create(@RequestBody LibraryOrderForm form) {
    }

    @GetMapping
    public void page(LibraryOrderForm query) {
    }
}

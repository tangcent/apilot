package com.example.arrayfield;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/array-fields")
public class ArrayFieldController {

    @PostMapping("/save")
    public ArrayFieldForm save(@RequestBody ArrayFieldForm form) {
        return form;
    }

    @PostMapping("/names")
    public String[] names(@RequestBody String[] names) {
        return names;
    }

    @GetMapping("/items")
    public ArrayFieldItem[] items() {
        return new ArrayFieldItem[0];
    }
}

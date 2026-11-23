package com.example.mixscope;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/** Number is the class of this package, in Java and for the Kotlin classes beside it. */
@RestController
@RequestMapping("/java")
public class JavaController {

    @GetMapping("/dto")
    public KotlinDto dto() {
        return null;
    }

    @PostMapping("/save")
    public void save(@RequestBody KotlinDto dto) {
    }

    @GetMapping("/number")
    public Number number() {
        return null;
    }
}

package com.example.nested.web;

import com.example.nested.NestOther.*;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/nested-star")
public class NestStarController {

    @GetMapping("/star")
    public Detail star() {
        return null;
    }

    @GetMapping("/fully-qualified")
    public com.example.nested.NestHolder.Detail fullyQualified() {
        return null;
    }
}

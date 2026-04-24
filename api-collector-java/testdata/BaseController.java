package com.example.demo.controller;

import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/api/base")
public class BaseController {

    @GetMapping("/health")
    public String healthCheck() {
        return "OK";
    }
}

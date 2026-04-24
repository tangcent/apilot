package com.example.demo.controller;

import org.springframework.web.bind.annotation.*;
import com.example.demo.model.Result;

@RestController
public class GenericBaseController<R> {

    @GetMapping("/info")
    public Result<R> getInfo() {
        return new Result<>();
    }
}

package com.example.controller;

import org.springframework.web.bind.annotation.*;
import com.example.model.Result;

@RestController
public class GenericBaseController<R> {

    @GetMapping("/info")
    public Result<R> getInfo() {
        return new Result<>();
    }
}

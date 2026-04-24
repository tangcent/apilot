package com.example.demo.controller;

import org.springframework.web.bind.annotation.*;
import com.example.demo.model.Result;
import com.example.demo.model.PageResult;

@RestController
public class BaseCrudController<Req, Res> {

    @PostMapping
    public Result<Res> create(@RequestBody Req request) {
        return new Result<>();
    }

    @GetMapping("/{id}")
    public Result<Res> getById(@PathVariable Long id) {
        return new Result<>();
    }

    @GetMapping
    public PageResult<Res> list(@RequestParam(defaultValue = "0") int page,
                                @RequestParam(defaultValue = "10") int size) {
        return new PageResult<>();
    }

    @PutMapping("/{id}")
    public Result<Res> update(@PathVariable Long id, @RequestBody Req request) {
        return new Result<>();
    }

    @DeleteMapping("/{id}")
    public Result<Void> delete(@PathVariable Long id) {
        return new Result<>();
    }
}

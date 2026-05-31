package com.example.demo.controller;

import io.swagger.annotations.Api;
import io.swagger.annotations.ApiOperation;
import io.swagger.annotations.ApiParam;
import org.springframework.web.bind.annotation.*;

/**
 * Swagger 2 annotated controller
 */
@Api(tags = "Swagger2")
@RestController
@RequestMapping("/api/swagger2")
public class Swagger2Controller {

    @ApiOperation(value = "Create item", httpMethod = "POST")
    @RequestMapping("/create")
    public void create(
            @ApiParam("Item name") @RequestParam String name,
            @ApiParam(value = "Item description", required = false, defaultValue = "N/A") String desc) {
    }

    @ApiOperation(value = "Get item by id", notes = "Returns a single item")
    @GetMapping("/{id}")
    public void getById(@ApiParam("Item ID") @PathVariable Long id) {
    }

    @ApiOperation(value = "Search items")
    @RequestMapping("/search")
    public void search(@ApiParam("Search keyword") String keyword) {
    }
}

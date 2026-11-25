package com.example.nested;

import java.util.List;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/nested")
public class NestedController {

    @PostMapping
    public void create(@RequestBody NestedHolder holder) {
    }

    @GetMapping
    public void page(NestedHolder query) {
    }

    @GetMapping("/values")
    public void values(Order.Status orderStatus, Product.Status productStatus, Product.Status[] productStatuses,
            List<Order.Status> orderStatusList) {
    }
}

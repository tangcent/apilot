package com.example.controller;

import org.springframework.web.bind.annotation.*;
import com.example.model.CreateOrderReq;
import com.example.model.OrderVO;

@RestController
@RequestMapping("/api/orders")
public class OrderController extends BaseCrudController<CreateOrderReq, OrderVO> {

    @GetMapping("/search")
    public OrderVO searchByName(@RequestParam String name) {
        return new OrderVO();
    }
}

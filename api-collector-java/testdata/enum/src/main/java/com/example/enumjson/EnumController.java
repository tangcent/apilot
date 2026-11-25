package com.example.enumjson;

import java.util.List;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/enums")
public class EnumController {

    @PostMapping
    public EnumEnvelope<OrderState> create(@RequestBody EnumForm form) {
        return null;
    }

    @GetMapping
    public void page(EnumForm query) {
    }

    @GetMapping("/state")
    public OrderState state() {
        return null;
    }

    @GetMapping("/states")
    public List<OrderState> states() {
        return null;
    }

    @GetMapping("/channel")
    public ChannelCode channel() {
        return null;
    }
}

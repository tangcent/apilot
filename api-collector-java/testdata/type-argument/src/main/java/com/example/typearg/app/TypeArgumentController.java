package com.example.typearg.app;

import com.example.typearg.dto.Item;
import com.example.typearg.lib.Box;
import com.example.typearg.lib.Holder;
import com.example.typearg.lib.Outer;
import java.util.List;
import java.util.Map;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/type-args")
public class TypeArgumentController {

    @PostMapping("/box")
    public Box<Item> box(@RequestBody Box<Item> body) {
        return body;
    }

    @GetMapping("/outer")
    public Outer<Item> outer() {
        return null;
    }

    @GetMapping("/outer-int")
    public Outer<Integer> outerInt() {
        return null;
    }

    @GetMapping("/box-list")
    public Box<List<Item>> boxList() {
        return null;
    }

    @GetMapping("/box-array")
    public Box<Item[]> boxArray() {
        return null;
    }

    @GetMapping("/box-map")
    public Box<Map<String, Item>> boxMap() {
        return null;
    }

    @GetMapping("/box-bounded")
    public Box<? extends Item> boxBounded() {
        return null;
    }

    @GetMapping("/holder")
    public Holder holder() {
        return null;
    }
}

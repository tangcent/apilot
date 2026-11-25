package com.example.jdkidentity.app;

import com.example.jdkidentity.lib.Box;
import com.example.jdkidentity.lib.Holder;
import com.example.jdkidentity.lib.Outer;
import java.util.Date;
import java.util.List;
import java.util.Map;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/** JDK classes as the arguments of the library's generic classes. */
@RestController
@RequestMapping("/jdk-args")
public class JdkArgumentController {

    @PostMapping("/box-string")
    public Box<String> boxString(@RequestBody Box<String> body) {
        return body;
    }

    @GetMapping("/box-integer")
    public Box<Integer> boxInteger() {
        return null;
    }

    @GetMapping("/box-date")
    public Box<Date> boxDate() {
        return null;
    }

    @GetMapping("/box-list")
    public Box<List<String>> boxList() {
        return null;
    }

    @GetMapping("/box-array")
    public Box<String[]> boxArray() {
        return null;
    }

    @GetMapping("/box-map")
    public Box<Map<String, Integer>> boxMap() {
        return null;
    }

    @GetMapping("/outer-string")
    public Outer<String> outerString() {
        return null;
    }

    @GetMapping("/outer-date")
    public Outer<Date> outerDate() {
        return null;
    }

    @GetMapping("/holder")
    public Holder holder() {
        return null;
    }
}

package com.example.jdkidentity.own;

import com.example.jdkidentity.lib.BaseController;
import com.example.jdkidentity.lib.Box;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/** String and Integer are the classes of this package, here and as arguments. */
@RestController
@RequestMapping("/jdk-own")
public class OwnController {

    @GetMapping("/box-string")
    public Box<String> boxString() {
        return null;
    }

    @GetMapping("/box-integer")
    public Box<Integer> boxInteger() {
        return null;
    }

    @GetMapping("/void")
    public Void nothing() {
        return null;
    }
}

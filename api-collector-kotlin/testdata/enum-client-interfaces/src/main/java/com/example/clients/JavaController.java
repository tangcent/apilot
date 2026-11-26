package com.example.clients;

import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/java")
public class JavaController implements MvcApi {
    @PostMapping("/create")
    public void create(@RequestBody Form form) {
    }

    @Override
    public String inherited() {
        return "";
    }
}

class Form {
    public Status status;
    public Plain plain;
}

enum Status implements Coded {
    A,
    B;

    public String getCode() {
        return "x";
    }
}

enum Plain {
    A,
    B
}

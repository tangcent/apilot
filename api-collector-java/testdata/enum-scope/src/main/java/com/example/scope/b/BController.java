package com.example.scope.b;

import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/scope/b")
public class BController {

    @PostMapping
    public void createB(@RequestBody Holder holder) {
    }
}

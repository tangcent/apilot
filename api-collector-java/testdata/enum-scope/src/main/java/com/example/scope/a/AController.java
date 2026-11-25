package com.example.scope.a;

import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/scope/a")
public class AController {

    @PostMapping
    public void createA(@RequestBody Holder holder) {
    }
}

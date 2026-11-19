package com.example.jdkscalar;

import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/jdk-scalars")
public class JdkScalarController {

    @PostMapping("/save")
    public JdkScalarForm save(@RequestBody JdkScalarForm form) {
        return form;
    }

    @PostMapping("/shorts")
    public short[] shorts(@RequestBody Short[] values) {
        return new short[0];
    }

    @PostMapping("/count")
    public java.lang.Short count(@RequestBody short value) {
        return value;
    }
}

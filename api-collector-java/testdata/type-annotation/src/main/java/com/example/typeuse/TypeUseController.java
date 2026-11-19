package com.example.typeuse;

import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/type-use")
public class TypeUseController {

    @PostMapping("/save")
    public TypeUseForm save(@RequestBody TypeUseForm form) {
        return form;
    }
}

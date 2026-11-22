package com.example.optionalscope.web;

import com.example.optionalscope.own.*;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/** Optional is the class of the package imported on demand. */
@RestController
@RequestMapping("/optional-scope")
public class DemandController {

    @GetMapping("/demand")
    public String demand(Optional<String> filter) {
        return "";
    }
}

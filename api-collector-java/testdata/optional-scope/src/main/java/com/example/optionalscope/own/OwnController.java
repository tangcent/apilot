package com.example.optionalscope.own;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/** Optional is the class of this package. */
@RestController
@RequestMapping("/optional-scope")
public class OwnController {

    @GetMapping("/own")
    public String own(Optional<String> filter) {
        return "";
    }
}

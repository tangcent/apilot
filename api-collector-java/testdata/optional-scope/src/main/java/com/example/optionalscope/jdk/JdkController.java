package com.example.optionalscope.jdk;

import java.util.Optional;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

/** Optional is java.util.Optional, imported by name. */
@RestController
@RequestMapping("/optional-scope")
public class JdkController {

    @GetMapping("/jdk")
    public String search(Optional<String> filter) {
        return "";
    }

    @PostMapping("/jdk-upload")
    public String upload(Optional<MultipartFile> avatar) {
        return "";
    }
}

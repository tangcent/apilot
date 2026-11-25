package com.example.uploadidentity.own;

import java.util.List;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

/** MultipartFile, HttpMethod, Locale, Model and Sort are the classes of this package. */
@RestController
@RequestMapping("/own")
public class OwnController {

    @PostMapping("/single")
    public String single(MultipartFile file) {
        return "";
    }

    @PostMapping("/optional")
    public String optional(java.util.Optional<MultipartFile> file) {
        return "";
    }

    @PostMapping("/list")
    public String list(List<MultipartFile> files) {
        return "";
    }

    @GetMapping("/names")
    public String names(HttpMethod method, Model model, Sort sort, Locale locale) {
        return "";
    }

    @GetMapping("/optional-param")
    public String optionalParam(@RequestParam("a") Optional<String> a) {
        return "";
    }

    @GetMapping("/own-optional")
    public String ownOptional(Optional<Model> model) {
        return "";
    }
}

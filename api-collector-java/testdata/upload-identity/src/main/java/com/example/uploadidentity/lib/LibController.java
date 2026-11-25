package com.example.uploadidentity.lib;

import java.util.List;
import java.util.Optional;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RequestPart;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

/** Spring's MultipartFile, by an import of the class, held by an Optional or not. */
@RestController
@RequestMapping("/lib")
public class LibController {

    @PostMapping("/single")
    public String single(MultipartFile file) {
        return "";
    }

    @PostMapping("/optional")
    public String optional(Optional<MultipartFile> avatar) {
        return "";
    }

    @PostMapping("/list")
    public String list(List<MultipartFile> files, MultipartFile[] docs) {
        return "";
    }

    @GetMapping("/optional-param")
    public String optionalParam(@RequestParam("a") Optional<String> a) {
        return "";
    }

    @PostMapping("/annotated")
    public String annotated(@RequestPart("cover") Optional<MultipartFile> cover,
                            @RequestParam("parts") List<MultipartFile> parts) {
        return "";
    }
}

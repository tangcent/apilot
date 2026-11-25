package com.example.uploadidentity.demand;

import java.util.*;
import org.springframework.web.bind.annotation.*;
import org.springframework.web.multipart.*;

/** The class comes from an import on demand of its package. */
@RestController
@RequestMapping("/demand")
public class DemandController {

    @PostMapping("/single")
    public String single(MultipartFile file, Optional<MultipartFile> avatar) {
        return "";
    }

    @PostMapping("/qualified")
    public String qualified(org.springframework.web.multipart.MultipartFile file) {
        return "";
    }
}

package com.example.nesteddep;

import com.example.nestlib.NestPage;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/nested-library")
public class NestLibraryController {

    @GetMapping("/sort")
    public NestPage.Sort sort() {
        return null;
    }

    @GetMapping("/footer")
    public com.example.nestlib.NestPage.Footer footer() {
        return null;
    }

    @GetMapping("/page")
    public NestPage<NestRow> page() {
        return null;
    }

    @GetMapping("/missing")
    public NestPage.Missing missing() {
        return null;
    }
}

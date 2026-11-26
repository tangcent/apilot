package com.example.pkgs.app;

import java.util.List;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/pkgs")
public class PackagesController {

    @PostMapping
    public void create(@RequestBody PackagesForm form) {
    }

    @GetMapping("/direct")
    public Status direct() {
        return null;
    }

    @GetMapping("/list")
    public List<Status> list() {
        return null;
    }
}

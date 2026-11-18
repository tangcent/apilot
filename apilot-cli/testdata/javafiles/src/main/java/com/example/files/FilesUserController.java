package com.example.files;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/files/users")
public class FilesUserController {

    /**
     * Page of users.
     */
    @GetMapping
    public FilesPage<FilesUser> list() {
        return null;
    }

    /**
     * One user.
     */
    @GetMapping("/{id}")
    public FilesUser get(@PathVariable Long id) {
        return null;
    }
}

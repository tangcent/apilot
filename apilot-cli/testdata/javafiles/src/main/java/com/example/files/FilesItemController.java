package com.example.files;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/files/items")
public class FilesItemController {

    /**
     * Items in stock.
     */
    @GetMapping
    public FilesPage<FilesUser> list() {
        return null;
    }
}

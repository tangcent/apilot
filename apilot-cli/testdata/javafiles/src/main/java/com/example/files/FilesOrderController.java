package com.example.files;

import java.util.List;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/files/orders")
public class FilesOrderController {

    /**
     * Orders of the current user.
     */
    @GetMapping
    public List<FilesOrderRequest> list() {
        return null;
    }

    /**
     * Place an order.
     */
    @PostMapping
    public FilesPage<FilesOrderRequest> create(@RequestBody FilesOrderRequest request) {
        return null;
    }
}

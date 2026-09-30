package com.example.demo.controller;

import org.springframework.web.bind.annotation.*;
import org.springframework.http.ResponseEntity;
import com.example.demo.model.User;

/**
 * Header declaration scenarios
 */
@RestController
@RequestMapping("/api/headers")
public class HeaderController {

    /**
     * Echo the authenticated user
     */
    @GetMapping("/me")
    public ResponseEntity<User> currentUser(@RequestHeader("Authorization") String authorization) {
        // Implementation here
        return ResponseEntity.ok(new User(1L, "Test User"));
    }

    /**
     * Report the negotiated API version
     */
    @GetMapping("/version")
    public ResponseEntity<String> version(
            @RequestHeader(value = "X-Api-Version", required = false) String apiVersion) {
        // Implementation here
        return ResponseEntity.ok(apiVersion);
    }
}

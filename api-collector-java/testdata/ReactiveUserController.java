package com.example.demo.controller;

import org.springframework.web.bind.annotation.*;
import com.example.demo.model.PageResult;
import com.example.demo.model.Result;
import com.example.demo.model.User;
import reactor.core.publisher.Flux;
import reactor.core.publisher.Mono;

/**
 * Reactive user API: handlers return reactor publishers instead of the
 * payload, so the exported schema must be the type argument.
 */
@RestController
@RequestMapping("/api/reactive/users")
public class ReactiveUserController {

    /**
     * Get user by ID
     */
    @GetMapping("/{id}")
    public Mono<User> getUser(@PathVariable Long id) {
        return Mono.empty();
    }

    /**
     * Get user profile
     */
    @GetMapping("/profile")
    public Mono<Result<User>> getProfile() {
        return Mono.empty();
    }

    /**
     * List users
     */
    @GetMapping
    public Mono<Result<PageResult<User>>> listUsers(
            @RequestParam(defaultValue = "0") int page,
            @RequestParam(defaultValue = "10") int size) {
        return Mono.empty();
    }

    /**
     * Delete user
     */
    @DeleteMapping("/{id}")
    public Mono<Result<Void>> deleteUser(@PathVariable Long id) {
        return Mono.empty();
    }

    /**
     * Stream users
     */
    @GetMapping("/stream")
    public Flux<User> streamUsers() {
        return Flux.empty();
    }
}

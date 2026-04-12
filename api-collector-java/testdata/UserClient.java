package com.example.demo.client;

import org.springframework.cloud.openfeign.FeignClient;
import org.springframework.web.bind.annotation.*;
import com.example.demo.model.User;
import java.util.List;

/**
 * Feign client for user-service (Spring Cloud OpenFeign style)
 */
@FeignClient(name = "user-service")
public interface UserClient {

    @GetMapping("/api/users/{id}")
    User getUser(@PathVariable Long id);

    @GetMapping("/api/users")
    List<User> listUsers(@RequestParam(required = false) int page,
                         @RequestParam(required = false) int size);

    @PostMapping("/api/users")
    User createUser(@RequestBody User user);

    @DeleteMapping("/api/users/{id}")
    void deleteUser(@PathVariable Long id);
}

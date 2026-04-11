package com.example;

import org.springframework.web.bind.annotation.*;
import org.springframework.http.ResponseEntity;

@RestController
@RequestMapping("/users")
public class UserController {

    @GetMapping
    public ResponseEntity<?> listUsers(@RequestParam(required = false) String name,
                                       @RequestParam(defaultValue = "user") String role) {
        return ResponseEntity.ok().body("{\"users\": []}");
    }

    @PostMapping
    public ResponseEntity<?> createUser(@RequestBody CreateUserReq req) {
        return ResponseEntity.status(201).body(req);
    }

    @GetMapping("/{id}")
    public ResponseEntity<?> getUser(@PathVariable String id) {
        return ResponseEntity.ok().body("{\"id\": \"" + id + "\"}");
    }

    @PutMapping("/{id}")
    public ResponseEntity<?> updateUser(@PathVariable String id, @RequestBody UpdateUserReq req) {
        return ResponseEntity.ok().body(req);
    }

    @DeleteMapping("/{id}")
    public ResponseEntity<?> deleteUser(@PathVariable String id) {
        return ResponseEntity.noContent().build();
    }

    @PatchMapping("/{id}")
    public ResponseEntity<?> patchUser(@PathVariable String id, 
                                       @RequestParam(defaultValue = "unknown") String name) {
        return ResponseEntity.ok().body("{\"id\": \"" + id + "\"}");
    }
}

class CreateUserReq {
    private String name;
    private String email;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
}

class UpdateUserReq {
    private String name;
    private String email;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
}

package com.example;

import java.util.List;

/**
 * A plain service with no HTTP annotations. The Java collector parses this file
 * without complaint and extracts zero endpoints, which is exactly the shape that
 * used to exit 0 with an empty document (issue #161).
 */
public class PlainService {

    private final List<String> roles;

    public PlainService(List<String> roles) {
        this.roles = roles;
    }

    public String greet(String name) {
        return "Hello, " + name;
    }

    public boolean canManage() {
        return roles.contains("admin");
    }
}

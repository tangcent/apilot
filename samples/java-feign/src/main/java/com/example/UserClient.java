package com.example;

import feign.Param;
import feign.RequestLine;

public interface UserClient {

    @RequestLine("GET /users?name={name}&role={role}")
    String listUsers(@Param("name") String name, @Param("role") String role);

    @RequestLine("POST /users")
    String createUser(CreateUserReq req);

    @RequestLine("GET /users/{id}")
    String getUser(@Param("id") String id);

    @RequestLine("PUT /users/{id}")
    String updateUser(@Param("id") String id, UpdateUserReq req);

    @RequestLine("DELETE /users/{id}")
    void deleteUser(@Param("id") String id);

    @RequestLine("PATCH /users/{id}?name={name}")
    String patchUser(@Param("id") String id, @Param("name") String name);
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

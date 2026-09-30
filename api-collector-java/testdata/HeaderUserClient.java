package com.example.demo.client;

import org.springframework.cloud.openfeign.FeignClient;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestHeader;

/**
 * Header declaration scenarios
 */
@FeignClient(name = "header-user")
public interface HeaderUserClient {

    /**
     * Echo the session token
     */
    @GetMapping("/api/headers/session")
    String session(@RequestHeader("X-Session-Token") String sessionToken);
}

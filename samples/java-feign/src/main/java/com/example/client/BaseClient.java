package com.example.client;

import feign.Param;
import feign.RequestLine;
import com.example.model.Result;

public interface BaseClient {
    @RequestLine("GET /info")
    Result<String> getInfo();

    @RequestLine("GET /health")
    Result<String> health();
}

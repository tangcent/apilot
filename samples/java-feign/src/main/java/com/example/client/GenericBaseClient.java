package com.example.client;

import feign.Param;
import feign.RequestLine;
import com.example.model.Result;

public interface GenericBaseClient<R> {
    @RequestLine("GET /info")
    Result<R> getInfo();
}

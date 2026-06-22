package com.example.client;

import feign.Param;
import feign.RequestLine;
import com.example.model.Result;
import com.example.model.PageResult;

public interface BaseCrudClient<Req, Res> {
    @RequestLine("POST")
    Result<Res> create(Req request);

    @RequestLine("GET /{id}")
    Result<Res> getById(@Param("id") Long id);

    @RequestLine("GET?page={page}&size={size}")
    PageResult<Res> list(@Param("page") int page, @Param("size") int size);

    @RequestLine("PUT /{id}")
    Result<Res> update(@Param("id") Long id, Req request);

    @RequestLine("DELETE /{id}")
    void delete(@Param("id") Long id);
}

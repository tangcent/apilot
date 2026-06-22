package com.example.controller;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import com.example.model.Result;

public class BaseController {
    @GET
    @Path("/info")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<String> getInfo() {
        return Result.success("info");
    }

    @GET
    @Path("/health")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<String> health() {
        return Result.success("ok");
    }
}

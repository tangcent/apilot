package com.example.controller;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import com.example.model.Result;

public class GenericBaseController<R> {
    @GET
    @Path("/info")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<R> getInfo() {
        return new Result<>();
    }
}

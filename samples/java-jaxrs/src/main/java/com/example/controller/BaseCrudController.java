package com.example.controller;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import com.example.model.Result;
import com.example.model.PageResult;

public abstract class BaseCrudController<Req, Res> {
    @POST
    @Consumes(MediaType.APPLICATION_JSON)
    @Produces(MediaType.APPLICATION_JSON)
    public Result<Res> create(Req request) {
        return Result.success(null);
    }

    @GET
    @Path("/{id}")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<Res> getById(@PathParam("id") Long id) {
        return Result.success(null);
    }

    @GET
    @Produces(MediaType.APPLICATION_JSON)
    public PageResult<Res> list(@QueryParam("page") int page, @QueryParam("size") int size) {
        return new PageResult<>();
    }

    @PUT
    @Path("/{id}")
    @Consumes(MediaType.APPLICATION_JSON)
    @Produces(MediaType.APPLICATION_JSON)
    public Result<Res> update(@PathParam("id") Long id, Req request) {
        return Result.success(null);
    }

    @DELETE
    @Path("/{id}")
    public Response delete(@PathParam("id") Long id) {
        return Response.noContent().build();
    }
}

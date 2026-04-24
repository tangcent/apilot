package com.example.demo.resource;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import java.util.List;

@Produces(MediaType.APPLICATION_JSON)
@Consumes(MediaType.APPLICATION_JSON)
public class BaseCrudResource<Req, Res> {

    @POST
    public Res create(Req request) {
        return null;
    }

    @GET
    @Path("/{id}")
    public Res getById(@PathParam("id") Long id) {
        return null;
    }

    @GET
    public List<Res> list(@QueryParam("page") int page,
                          @QueryParam("size") int size) {
        return null;
    }

    @PUT
    @Path("/{id}")
    public Res update(@PathParam("id") Long id, Req request) {
        return null;
    }

    @DELETE
    @Path("/{id}")
    public Response delete(@PathParam("id") Long id) {
        return Response.noContent().build();
    }
}

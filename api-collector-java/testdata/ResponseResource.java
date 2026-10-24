package com.example.demo.resource;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import com.example.demo.model.PageResult;
import com.example.demo.model.Result;
import com.example.demo.model.User;

/**
 * Resource whose response payload is built in the method body: the declared
 * type is Response, which says nothing about what comes back.
 */
@Path("/api/responses")
@Produces(MediaType.APPLICATION_JSON)
public class ResponseResource {

    /**
     * The entity is held in a local variable and wrapped in a generic result.
     */
    @GET
    @Path("/{id}")
    public Response get(@PathParam("id") Long id) {
        User user = new User();
        return Response.ok(Result.success(user)).build();
    }

    /**
     * The entity is attached with the builder's entity method.
     */
    @POST
    public Response create(User user) {
        User created = new User();
        return Response.status(201).entity(Result.success(created)).build();
    }

    /**
     * A paged payload: the wrapper takes the page element type from the body.
     */
    @GET
    @Path("/paged")
    public Response paged(@QueryParam("page") int page) {
        PageResult<User> users = new PageResult<>();
        return Response.ok(Result.success(users)).build();
    }

    /**
     * A response with no entity at all.
     */
    @DELETE
    @Path("/{id}")
    public Response delete(@PathParam("id") Long id) {
        return Response.noContent().build();
    }

    /**
     * The payload comes from another method, so its type is not written
     * anywhere in this body.
     */
    @PUT
    @Path("/{id}")
    public Response update(@PathParam("id") Long id, User user) {
        return Response.ok(load(id)).build();
    }

    private User load(Long id) {
        return new User();
    }
}

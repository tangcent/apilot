package com.example.demo.resource;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import com.example.demo.model.User;
import java.util.List;

/**
 * User management JAX-RS resource
 */
@Path("/api/users")
@Produces(MediaType.APPLICATION_JSON)
@Consumes(MediaType.APPLICATION_JSON)
public class UserResource {

    /**
     * Get user by ID
     */
    @GET
    @Path("/{id}")
    public User getUser(@PathParam("id") Long id) {
        return new User(id, "Test User");
    }

    /**
     * List all users
     */
    @GET
    public List<User> listUsers(
            @QueryParam("page") int page,
            @QueryParam("size") int size) {
        return List.of();
    }

    /**
     * Create new user
     */
    @POST
    public Response createUser(User user) {
        return Response.ok(user).build();
    }

    /**
     * Update user
     */
    @PUT
    @Path("/{id}")
    public Response updateUser(@PathParam("id") Long id, User user) {
        return Response.ok(user).build();
    }

    /**
     * Delete user
     */
    @DELETE
    @Path("/{id}")
    public Response deleteUser(@PathParam("id") Long id) {
        return Response.noContent().build();
    }
}

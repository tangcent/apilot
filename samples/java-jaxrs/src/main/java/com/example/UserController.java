package com.example;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import java.util.List;

@Path("/users")
public class UserController {

    @GET
    @Produces(MediaType.APPLICATION_JSON)
    public List<UserVO> listUsers(@QueryParam("name") String name,
                             @DefaultValue("user") @QueryParam("role") String role) {
        return List.of();
    }

    @POST
    @Consumes(MediaType.APPLICATION_JSON)
    @Produces(MediaType.APPLICATION_JSON)
    public UserVO createUser(CreateUserReq req) {
        return new UserVO();
    }

    @GET
    @Path("/{id}")
    @Produces(MediaType.APPLICATION_JSON)
    public UserVO getUser(@PathParam("id") String id) {
        return new UserVO();
    }

    @PUT
    @Path("/{id}")
    @Consumes(MediaType.APPLICATION_JSON)
    @Produces(MediaType.APPLICATION_JSON)
    public UserVO updateUser(@PathParam("id") String id, UpdateUserReq req) {
        return new UserVO();
    }

    @DELETE
    @Path("/{id}")
    public Response deleteUser(@PathParam("id") String id) {
        return Response.noContent().build();
    }

    @PATCH
    @Path("/{id}")
    @Produces(MediaType.APPLICATION_JSON)
    public UserVO patchUser(@PathParam("id") String id,
                             @DefaultValue("unknown") @QueryParam("name") String name) {
        return new UserVO();
    }
}

class CreateUserReq {
    private String name;
    private String email;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
}

class UpdateUserReq {
    private String name;
    private String email;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
}

class UserVO {
    private Long id;
    private String name;
    private String email;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getEmail() { return email; }
    public void setEmail(String email) { this.email = email; }
}

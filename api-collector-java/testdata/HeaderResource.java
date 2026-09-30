package com.example.demo.resource;

import jakarta.ws.rs.GET;
import jakarta.ws.rs.HeaderParam;
import jakarta.ws.rs.Path;
import jakarta.ws.rs.Produces;
import jakarta.ws.rs.core.MediaType;

/**
 * Header declaration scenarios
 */
@Path("/api/header-resources")
public class HeaderResource {

    /**
     * Echo the request id
     */
    @GET
    @Path("/request-id")
    @Produces(MediaType.APPLICATION_JSON)
    public String requestId(@HeaderParam("X-Request-Id") String requestId) {
        // Implementation here
        return requestId;
    }
}

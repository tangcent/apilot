package com.example.deprecated;

import javax.ws.rs.GET;
import javax.ws.rs.Path;

/**
 * Legacy resource.
 *
 * @deprecated replaced by OrderController
 */
@Path("/legacy-resource")
public class LegacyResource {

    /**
     * Ping.
     */
    @GET
    @Path("/ping")
    public String ping() {
        return "";
    }
}

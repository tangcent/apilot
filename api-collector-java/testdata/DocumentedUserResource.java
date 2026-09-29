package com.example.demo.resource;

import com.example.demo.model.DocumentedResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import javax.ws.rs.DefaultValue;
import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.QueryParam;

/**
 * Documented JAX-RS API
 */
@Path("/api/docs")
public class DocumentedUserResource {

    /**
     * JavaDoc fallback summary.
     *
     * @param id fallback id
     * @param status fallback status
     */
    @Operation(summary = "Fetch documented item", description = "Returns a documented item by id")
    @GET
    @Path("/{id}")
    public DocumentedResponse getDocumented(
            @Parameter(description = "Documented item id", example = "42", required = true)
            @PathParam("id") Long id,
            @DefaultValue("active") @QueryParam("status") String status) {
        return new DocumentedResponse();
    }

    /**
     * Search documented items.
     *
     * @param keyword search keyword
     */
    @GET
    @Path("/search")
    public DocumentedResponse searchDocumented(
            @QueryParam("keyword") String keyword,
            @DefaultValue("10") @QueryParam("limit") Integer limit) {
        return new DocumentedResponse();
    }
}

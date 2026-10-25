package com.example.demo.resource;

import javax.ws.rs.Consumes;
import javax.ws.rs.FormParam;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;

/**
 * Multipart form scenarios on the JAX-RS side: the class declares the media
 * types its methods inherit, and a file part is a form parameter.
 */
@Path("/api/uploads")
@Consumes(MediaType.MULTIPART_FORM_DATA)
@Produces(MediaType.APPLICATION_JSON)
public class UploadResource {

    @POST
    public String upload(
            @FormParam("file") MultipartFile file,
            @FormParam("description") String description) {
        return file.getOriginalFilename();
    }

    @POST
    @Path("/plain")
    @Consumes(MediaType.APPLICATION_FORM_URLENCODED)
    @Produces(MediaType.TEXT_PLAIN)
    public String uploadPlain(@FormParam("note") String note) {
        return note;
    }
}

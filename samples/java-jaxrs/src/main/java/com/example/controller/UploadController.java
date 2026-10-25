package com.example.controller;

import javax.ws.rs.Consumes;
import javax.ws.rs.FormParam;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import com.example.model.Result;

/**
 * Multipart upload: the class declares the media types its methods inherit,
 * and a file part is a form parameter — never a query parameter.
 */
@Path("/api/uploads")
@Consumes(MediaType.MULTIPART_FORM_DATA)
@Produces(MediaType.APPLICATION_JSON)
public class UploadController {

    @POST
    public Result<String> upload(
            @FormParam("file") MultipartFile file,
            @FormParam("description") String description) {
        return Result.success(file.getOriginalFilename());
    }
}

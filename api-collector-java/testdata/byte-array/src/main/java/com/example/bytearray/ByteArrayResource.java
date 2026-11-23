package com.example.bytearray;

import javax.ws.rs.Consumes;
import javax.ws.rs.POST;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;

@Path("/byte-arrays/jaxrs")
@Produces(MediaType.APPLICATION_JSON)
@Consumes(MediaType.APPLICATION_JSON)
public class ByteArrayResource {

    @POST
    @Path("/outer")
    public ByteArrayOuter<byte [ ]> outer(ByteArrayOuter<byte [ ]> body) {
        return body;
    }

    @POST
    @Path("/sub")
    public ByteArraySub sub(ByteArraySub body) {
        return body;
    }
}

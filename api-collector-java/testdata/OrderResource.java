package com.example.demo.resource;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import java.util.List;
import com.example.demo.model.User;
import com.example.demo.model.CreateOrderReq;
import com.example.demo.model.OrderVO;

@Path("/api/orders")
public class OrderResource extends BaseCrudResource<CreateOrderReq, OrderVO> {

    @GET
    @Path("/search")
    public List<OrderVO> search(@QueryParam("keyword") String keyword) {
        return null;
    }

    @POST
    @Path("/batch")
    public Response batchCreate(List<CreateOrderReq> requests) {
        return Response.ok().build();
    }
}

package com.example.clients

import com.fasterxml.jackson.annotation.JsonValue
import feign.Headers
import feign.Param
import feign.RequestLine
import javax.ws.rs.GET
import javax.ws.rs.Path
import javax.ws.rs.PathParam
import org.springframework.cloud.openfeign.FeignClient
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PathVariable

/** A Feign client written with @RequestLine. */
interface LineClient {
    @RequestLine("GET /line/{id}")
    @Headers("Accept: application/json")
    fun line(@Param("id") id: Long): String

    @RequestLine("GET /line/all")
    fun all(): List<String>
}

/** A Spring Cloud OpenFeign client. */
@FeignClient(name = "cloud")
interface CloudClient {
    @GetMapping("/cloud/{id}")
    @Headers("Accept: application/json")
    fun cloud(@PathVariable("id") id: Long): String
}

/** A JAX-RS resource. */
@Path("/rs")
interface RsResource {
    @GET
    @Path("/{id}")
    fun rs(@PathParam("id") id: String): String
}

/** A Spring MVC contract a Java controller implements. */
interface MvcApi {
    @GetMapping("/inherited")
    @Audited
    fun inherited(): String
}

annotation class Audited

interface Coded {
    @get:JsonValue
    val code: String
}

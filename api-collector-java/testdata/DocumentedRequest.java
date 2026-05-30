package com.example.demo.model;

import io.swagger.v3.oas.annotations.media.Schema;

public class DocumentedRequest {
    /**
     * Name fallback
     */
    @Schema(description = "Requested name", example = "Ada", required = true)
    private String name;
}

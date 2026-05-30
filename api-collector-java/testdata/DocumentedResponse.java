package com.example.demo.model;

import io.swagger.v3.oas.annotations.media.Schema;

public class DocumentedResponse {
    /**
     * Display name fallback
     */
    @Schema(description = "Display name", example = "Ada", required = true)
    private String displayName;
}

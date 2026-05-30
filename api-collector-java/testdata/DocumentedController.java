package com.example.demo.controller;

import com.example.demo.model.DocumentedRequest;
import com.example.demo.model.DocumentedResponse;
import io.swagger.v3.oas.annotations.Operation;
import io.swagger.v3.oas.annotations.Parameter;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

/**
 * Documented Spring API
 */
@RestController
@RequestMapping("/api/docs")
public class DocumentedController {

    /**
     * JavaDoc fallback summary.
     *
     * @param id fallback id
     * @param status fallback status
     * @param request fallback request
     */
    @Operation(summary = "Fetch documented item", description = "Returns a documented item by id")
    @GetMapping("/{id}")
    public ResponseEntity<DocumentedResponse> getDocumented(
            @Parameter(description = "Documented item id", example = "42", required = true)
            @PathVariable Long id,
            @RequestParam(defaultValue = "active") String status,
            @RequestBody DocumentedRequest request) {
        return ResponseEntity.ok(new DocumentedResponse());
    }
}

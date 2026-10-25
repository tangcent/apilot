package com.example.demo.controller;

import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RequestPart;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

/**
 * Multipart upload scenarios: a MultipartFile argument is a form part, never a
 * query parameter, and the mapping annotation declares the media types.
 */
@RestController
public class UploadController {

    @PostMapping(value = "/api/uploads",
            consumes = MediaType.MULTIPART_FORM_DATA,
            produces = MediaType.APPLICATION_JSON)
    public ResponseEntity<String> upload(
            @RequestParam("file") MultipartFile file,
            @RequestParam(value = "description", required = false) String description) {
        return ResponseEntity.ok(file.getOriginalFilename());
    }

    @PostMapping(value = "/api/uploads/parts", consumes = "multipart/form-data")
    public ResponseEntity<String> uploadParts(
            @RequestPart("avatar") MultipartFile avatar,
            @RequestPart("bio") String bio) {
        return ResponseEntity.ok(avatar.getOriginalFilename());
    }
}

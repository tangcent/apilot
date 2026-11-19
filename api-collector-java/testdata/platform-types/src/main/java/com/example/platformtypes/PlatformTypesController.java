package com.example.platformtypes;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.node.ObjectNode;
import java.time.DayOfWeek;
import java.util.Currency;
import java.util.Optional;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.multipart.MultipartFile;

// The same handlers as api-collector-kotlin/testdata/platform-types/PlatformTypesController.kt.
@RestController
@RequestMapping("/platform-types")
public class PlatformTypesController {

    @PostMapping("/save")
    public PlatformTypesForm save(@RequestBody PlatformTypesForm form) {
        return form;
    }

    @PostMapping("/wildcard")
    public PlatformWildcardForm wildcard(@RequestBody PlatformWildcardForm form) {
        return form;
    }

    @GetMapping("/search")
    public String search(Optional<String> keyword, @RequestParam Optional<Integer> page,
                         Optional<PlatformItem> filter, Currency currency, DayOfWeek day) {
        return "";
    }

    @PostMapping("/upload")
    public String upload(@RequestParam("file") MultipartFile file, PlatformUploadForm form,
                         Optional<MultipartFile> avatar) {
        return "";
    }

    @PostMapping("/nodes")
    public JsonNode nodes(@RequestBody ObjectNode body) {
        return body;
    }

    @PostMapping("/names")
    public java.util.List<PlatformItem> names(@RequestBody java.util.Set<String> names) {
        return null;
    }

    @PostMapping("/maybe")
    public Optional<PlatformItem> maybe(@RequestBody Optional<PlatformItem> item) {
        return item;
    }
}

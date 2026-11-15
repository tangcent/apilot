package com.example.optional;

import java.util.Optional;

import javax.validation.constraints.NotNull;

import org.springframework.lang.Nullable;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/search")
public class SearchController {

    @GetMapping
    public void search(@RequestParam String keyword,
                       @RequestParam Optional<String> tag,
                       @RequestParam(defaultValue = "20") int size,
                       @RequestParam @Nullable String cursor,
                       @RequestParam(required = true) Optional<String> region,
                       @RequestParam @NotNull Optional<Long> ownerId,
                       @RequestHeader("X-Trace") @Nullable String trace,
                       @RequestHeader("X-Tenant") String tenant) {
    }
}

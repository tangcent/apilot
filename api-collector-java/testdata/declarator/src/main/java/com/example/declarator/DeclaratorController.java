package com.example.declarator;

import javax.validation.constraints.NotNull;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/declarators")
public class DeclaratorController {

    @PostMapping("/save")
    public DeclaratorForm save(@RequestBody DeclaratorForm form) {
        return form;
    }

    /** The names on the wire are constants declared in one statement. */
    @GetMapping("/search")
    public String[] search(@RequestParam(DeclaratorKeys.PAGE) int page,
                           @RequestParam(DeclaratorKeys.SIZE) int size,
                           @RequestParam(required = false) String... ids) {
        return ids;
    }

    @GetMapping("/search-array")
    public String[] searchArray(@RequestParam(required = false) String[] ids) {
        return ids;
    }

    @GetMapping("/implicit")
    public String[] implicit(String... tags) {
        return tags;
    }

    /** The spaces around the ellipsis and the annotation are not part of the type. */
    @GetMapping("/spaced")
    public String[] spaced(String ... names) {
        return names;
    }

    @GetMapping("/annotated")
    public String[] annotated(@NotNull String... names) {
        return names;
    }

    @GetMapping("/annotated-spaced")
    public String[] annotatedSpaced(@NotNull String ... names) {
        return names;
    }

    @PostMapping("/items")
    public DeclaratorItem[] items(@RequestBody DeclaratorItem... items) {
        return items;
    }

    @PostMapping("/items-array")
    public DeclaratorItem[] itemsArray(@RequestBody DeclaratorItem[] items) {
        return items;
    }
}

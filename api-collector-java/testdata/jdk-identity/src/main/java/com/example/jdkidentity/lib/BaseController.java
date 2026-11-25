package com.example.jdkidentity.lib;

import java.util.List;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;

/** A generic controller of the library, in a package that declares String, Integer and Date. */
public abstract class BaseController<T> {

    @GetMapping("/one")
    public T one() {
        return null;
    }

    @GetMapping("/all")
    public List<T> all() {
        return null;
    }

    @PostMapping("/save")
    public T save(@RequestBody T body) {
        return body;
    }

    @GetMapping("/search")
    public List<T> search(T query) {
        return null;
    }
}

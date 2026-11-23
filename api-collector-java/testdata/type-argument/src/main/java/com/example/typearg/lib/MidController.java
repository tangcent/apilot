package com.example.typearg.lib;

import java.util.List;
import org.springframework.web.bind.annotation.GetMapping;

/** Names its own parameter T, like the controller it extends. */
public abstract class MidController<T> extends BaseController<List<T>> {

    @GetMapping("/mid")
    public T mid() {
        return null;
    }
}

package com.example.bytearray;

import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;

public abstract class ByteArrayBaseController<T> {

    @PostMapping("/echo")
    public ByteArrayResult<T> echo(@RequestBody T body) {
        return null;
    }
}

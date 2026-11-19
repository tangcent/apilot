package com.example.bytearray;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/byte-arrays")
public class ByteArrayController {

    @PostMapping("/save")
    public ByteArrayForm save(@RequestBody ByteArrayForm form) {
        return form;
    }

    @PostMapping("/blob")
    public byte[] blob(@RequestBody byte[] blob) {
        return blob;
    }

    @GetMapping("/query")
    public String query(@RequestParam byte[] data, @RequestParam Byte[] boxed) {
        return "";
    }

    @GetMapping("/bound")
    public String bound(ByteArrayQuery query) {
        return "";
    }
}

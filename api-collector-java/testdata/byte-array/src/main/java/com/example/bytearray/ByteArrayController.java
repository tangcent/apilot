package com.example.bytearray;

import java.util.List;
import java.util.Map;
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

    @PostMapping("/spaced")
    public Map<String, byte [ ]> spaced(@RequestBody List<byte []> blobs) {
        return null;
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

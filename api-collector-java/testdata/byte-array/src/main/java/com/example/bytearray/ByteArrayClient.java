package com.example.bytearray;

import org.springframework.cloud.openfeign.FeignClient;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;

@FeignClient(name = "blobs")
public interface ByteArrayClient {

    @PostMapping("/byte-arrays/feign/outer")
    ByteArrayOuter<byte [ ]> outer(@RequestBody ByteArrayOuter<byte [ ]> body);
}

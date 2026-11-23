package com.example.bytearray;

import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/byte-arrays/typed")
public class ByteArrayTypedController extends ByteArrayBaseController<byte [ ]> {
}

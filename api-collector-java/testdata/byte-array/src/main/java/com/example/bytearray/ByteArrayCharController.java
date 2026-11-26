package com.example.bytearray;

import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/byte-arrays/typed-chars")
public class ByteArrayCharController extends ByteArrayBaseController<char [ ]> {
}

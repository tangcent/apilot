package com.example.byteannotated;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonRawValue;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import org.springframework.web.bind.annotation.CookieValue;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RequestPart;
import org.springframework.web.bind.annotation.RestController;

class ByteAnnotatedForm {
    byte[] plain;
    char[] letters;
    @JsonSerialize(using = Nums.class) byte[] using;
    @JsonSerialize(converter = Conv.class) byte[] converter;
    @JsonSerialize(using = JsonSerializer.None.class) byte[] none;
    @JsonSerialize(contentUsing = Nums.class) byte[] contentSelf;
    @JsonSerialize(contentUsing = Nums.class) byte[][] chunks;
    @JsonSerialize(contentUsing = Nums.class) List<byte[]> blobs;
    @JsonSerialize(contentUsing = Nums.class) Map<String, byte[]> byName;
    @JsonRawValue byte[] raw;
    @JsonFormat(shape = JsonFormat.Shape.ARRAY) byte[] fmtArray;
    @JsonFormat(shape = JsonFormat.Shape.NUMBER) byte[] fmtNumber;
    @JsonFormat(shape = JsonFormat.Shape.STRING) byte[] fmtString;
    @JsonFormat(shape = JsonFormat.Shape.OBJECT) byte[] fmtObject;
    @JsonSerialize(using = Nums.class) char[] lettersUsing;
    byte[] nullable;
    Optional<byte[]> optional;
}

@JsonFormat(shape = JsonFormat.Shape.ARRAY)
class ByteAnnotatedClassFmt {
    byte[] data;
}

class ByteAnnotatedQuery {
    byte[] data;
    char[] letters;
}

interface ByteAnnotatedApi {
    @PostMapping("/default")
    default byte[] defaults(@RequestBody byte[] body) { return body; }
}

@RestController
@RequestMapping("/byte-annotated")
class ByteAnnotatedController implements ByteAnnotatedApi {
    @PostMapping("/form")
    ByteAnnotatedForm form(@RequestBody ByteAnnotatedForm body) { return body; }

    @PostMapping("/class-format")
    ByteAnnotatedClassFmt classFormat(@RequestBody ByteAnnotatedClassFmt body) { return body; }

    @PostMapping("/whole")
    byte[] whole(@RequestBody byte[] body) { return body; }

    @PostMapping("/entity")
    org.springframework.http.ResponseEntity<byte[]> entity(@RequestBody byte[] body) { return null; }

    @PostMapping("/mono")
    reactor.core.publisher.Mono<byte[]> mono(@RequestBody byte[] body) { return null; }

    @PostMapping("/flux")
    reactor.core.publisher.Flux<byte[]> flux(@RequestBody byte[] body) { return null; }

    @GetMapping("/query")
    String query(@RequestParam byte[] data) { return ""; }

    @GetMapping("/header")
    String header(@RequestHeader byte[] data) { return ""; }

    @GetMapping("/cookie")
    String cookie(@CookieValue byte[] data) { return ""; }

    @PostMapping("/part")
    String part(@RequestPart byte[] data) { return ""; }

    @GetMapping("/bound")
    String bound(ByteAnnotatedQuery query) { return ""; }
}

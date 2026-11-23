package com.example.bytearray;

import java.util.List;
import java.util.Map;
import javax.validation.constraints.NotNull;
import org.springframework.http.ResponseEntity;
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

    @PostMapping("/text")
    public char[] text(@RequestBody char[] text) {
        return text;
    }

    @PostMapping("/spaced")
    public Map<String, byte [ ]> spaced(@RequestBody List<byte []> blobs) {
        return null;
    }

    @PostMapping("/spaced-chars")
    public Map<String, char [ ]> spacedChars(@RequestBody List<char []> words) {
        return null;
    }

    @PostMapping("/outer-chars")
    public ByteArrayOuter<char [ ]> outerChars(@RequestBody ByteArrayOuter<char @NotNull [ ]> body) {
        return body;
    }

    @PostMapping("/outer-boxed-chars")
    public ByteArrayOuter<Character[]> outerBoxedChars(@RequestBody ByteArrayOuter<Character[]> body) {
        return body;
    }

    @PostMapping("/char-sub")
    public ByteArrayCharSub charSub(@RequestBody ByteArrayCharSub body) {
        return body;
    }

    @PostMapping("/outer")
    public ByteArrayOuter<byte [ ]> outer(@RequestBody ByteArrayOuter<byte @NotNull [ ]> body) {
        return body;
    }

    @PostMapping("/outer-boxed")
    public ByteArrayOuter<Byte[]> outerBoxed(@RequestBody ByteArrayOuter<Byte[]> body) {
        return body;
    }

    @PostMapping("/outer-chunks")
    public ByteArrayOuter<byte[][]> outerChunks(@RequestBody ByteArrayOuter<byte [ ][ ]> body) {
        return body;
    }

    @PostMapping("/outer-list")
    public ResponseEntity<ByteArrayOuter<List<byte [ ]>>> outerList(@RequestBody ByteArrayOuter<List<byte [ ]>> body) {
        return null;
    }

    @PostMapping("/outer-wildcard")
    public ByteArrayOuter<? extends byte[]> outerWildcard(@RequestBody ByteArrayOuter<? extends byte[]> body) {
        return body;
    }

    @PostMapping("/sub")
    public ByteArraySub sub(@RequestBody ByteArraySub body) {
        return body;
    }

    @PostMapping("/sub-boxed")
    public ByteArrayBoxedSub subBoxed(@RequestBody ByteArrayBoxedSub body) {
        return body;
    }

    @PostMapping("/open-sub")
    public ByteArrayOpenSub<byte [ ]> openSub(@RequestBody ByteArrayOpenSub<byte [ ]> body) {
        return body;
    }

    @PostMapping("/by-name")
    public Map<String, ByteArrayWrapper<byte [ ]>> byName(@RequestBody Map<String, ByteArrayWrapper<byte [ ]>> body) {
        return body;
    }

    @GetMapping("/query")
    public String query(@RequestParam byte[] data, @RequestParam Byte[] boxed,
            @RequestParam char[] letters, @RequestParam Character[] boxedLetters) {
        return "";
    }

    @GetMapping("/bound-outer")
    public String boundOuter(ByteArrayOuter<byte [ ]> query) {
        return "";
    }

    @GetMapping("/bound-outer-chars")
    public String boundOuterChars(ByteArrayOuter<char [ ]> query) {
        return "";
    }

    @GetMapping("/bound")
    public String bound(ByteArrayQuery query) {
        return "";
    }
}

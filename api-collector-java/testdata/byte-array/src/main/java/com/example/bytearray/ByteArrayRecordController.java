package com.example.bytearray;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

// A varargs component of a record is the array it is declared with, as a varargs
// parameter is: `byte... data` is a byte[] and `Byte... data` a Byte[]. A record
// takes one varargs component, so each has an explicit twin.
@RestController
@RequestMapping("/byte-records")
public class ByteArrayRecordController {

    @PostMapping("/explicit/bytes")
    public ExplicitBytesRecord explicitBytes(@RequestBody ExplicitBytesRecord form) {
        return form;
    }

    @PostMapping("/vararg/bytes")
    public VarargBytesRecord varargBytes(@RequestBody VarargBytesRecord form) {
        return form;
    }

    @GetMapping("/explicit-bound/bytes")
    public String explicitBoundBytes(ExplicitBytesRecord form) {
        return "";
    }

    @GetMapping("/vararg-bound/bytes")
    public String varargBoundBytes(VarargBytesRecord form) {
        return "";
    }

    @PostMapping("/explicit/chars")
    public ExplicitCharsRecord explicitChars(@RequestBody ExplicitCharsRecord form) {
        return form;
    }

    @PostMapping("/vararg/chars")
    public VarargCharsRecord varargChars(@RequestBody VarargCharsRecord form) {
        return form;
    }

    @GetMapping("/explicit-bound/chars")
    public String explicitBoundChars(ExplicitCharsRecord form) {
        return "";
    }

    @GetMapping("/vararg-bound/chars")
    public String varargBoundChars(VarargCharsRecord form) {
        return "";
    }

    @PostMapping("/explicit/boxed-bytes")
    public ExplicitBoxedBytesRecord explicitBoxedBytes(@RequestBody ExplicitBoxedBytesRecord form) {
        return form;
    }

    @PostMapping("/vararg/boxed-bytes")
    public VarargBoxedBytesRecord varargBoxedBytes(@RequestBody VarargBoxedBytesRecord form) {
        return form;
    }

    @GetMapping("/explicit-bound/boxed-bytes")
    public String explicitBoundBoxedBytes(ExplicitBoxedBytesRecord form) {
        return "";
    }

    @GetMapping("/vararg-bound/boxed-bytes")
    public String varargBoundBoxedBytes(VarargBoxedBytesRecord form) {
        return "";
    }

    @PostMapping("/explicit/boxed-chars")
    public ExplicitBoxedCharsRecord explicitBoxedChars(@RequestBody ExplicitBoxedCharsRecord form) {
        return form;
    }

    @PostMapping("/vararg/boxed-chars")
    public VarargBoxedCharsRecord varargBoxedChars(@RequestBody VarargBoxedCharsRecord form) {
        return form;
    }

    @GetMapping("/explicit-bound/boxed-chars")
    public String explicitBoundBoxedChars(ExplicitBoxedCharsRecord form) {
        return "";
    }

    @GetMapping("/vararg-bound/boxed-chars")
    public String varargBoundBoxedChars(VarargBoxedCharsRecord form) {
        return "";
    }

    @PostMapping("/explicit/names")
    public ExplicitNamesRecord explicitNames(@RequestBody ExplicitNamesRecord form) {
        return form;
    }

    @PostMapping("/vararg/names")
    public VarargNamesRecord varargNames(@RequestBody VarargNamesRecord form) {
        return form;
    }

    @GetMapping("/explicit-bound/names")
    public String explicitBoundNames(ExplicitNamesRecord form) {
        return "";
    }

    @GetMapping("/vararg-bound/names")
    public String varargBoundNames(VarargNamesRecord form) {
        return "";
    }

    @PostMapping("/explicit/ints")
    public ExplicitIntsRecord explicitInts(@RequestBody ExplicitIntsRecord form) {
        return form;
    }

    @PostMapping("/vararg/ints")
    public VarargIntsRecord varargInts(@RequestBody VarargIntsRecord form) {
        return form;
    }

    @GetMapping("/explicit-bound/ints")
    public String explicitBoundInts(ExplicitIntsRecord form) {
        return "";
    }

    @GetMapping("/vararg-bound/ints")
    public String varargBoundInts(VarargIntsRecord form) {
        return "";
    }
}

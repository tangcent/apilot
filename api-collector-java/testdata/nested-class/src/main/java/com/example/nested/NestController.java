package com.example.nested;

import java.util.List;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/nested")
public class NestController {

    @GetMapping("/holder")
    public NestHolder holder() {
        return null;
    }

    @GetMapping("/other")
    public NestOther other() {
        return null;
    }

    @GetMapping("/child")
    public NestChild child() {
        return null;
    }

    @GetMapping("/mid")
    public NestHolder.Mid mid() {
        return null;
    }

    @GetMapping("/detail")
    public NestHolder.Detail detail() {
        return null;
    }

    @PostMapping("/create")
    public NestEnvelope<NestOther.Detail> create(@RequestBody NestHolder.Detail body) {
        return null;
    }

    @GetMapping("/envelope")
    public NestEnvelope<NestHolder.Detail> envelope() {
        return null;
    }

    @GetMapping("/list")
    public List<NestOther.Detail> list() {
        return null;
    }
}

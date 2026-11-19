package com.example.ondemand;

import com.example.lib.model.*;
import com.example.lib.response.*;
import java.util.*;
import org.springframework.web.bind.annotation.*;

@RestController
@RequestMapping("/star-import")
public class StarImportController {

    @PostMapping("/create")
    public StarEnvelope<List<StarItem>> create(@RequestBody StarOrder order) {
        return null;
    }

    @GetMapping("/missing")
    public StarEnvelope<StarMissing> missing() {
        return null;
    }
}

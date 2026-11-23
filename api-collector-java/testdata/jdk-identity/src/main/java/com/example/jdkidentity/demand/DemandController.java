package com.example.jdkidentity.demand;

import com.example.jdkidentity.lib.BaseController;
import com.example.jdkidentity.lib.Box;
import java.util.*;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/** Date comes from an import on demand of java.util. */
@RestController
@RequestMapping("/jdk-demand")
public class DemandController {

    @GetMapping("/box-date")
    public Box<Date> boxDate() {
        return null;
    }

    @GetMapping("/box-uuid")
    public Box<UUID> boxUuid() {
        return null;
    }
}

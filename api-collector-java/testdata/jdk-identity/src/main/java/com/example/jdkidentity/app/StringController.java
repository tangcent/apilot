package com.example.jdkidentity.app;

import com.example.jdkidentity.lib.BaseController;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/jdk-strings")
public class StringController extends BaseController<String> {
}

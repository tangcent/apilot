package com.example.jdkidentity.own;

import com.example.jdkidentity.lib.BaseController;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/jdk-own-strings")
public class OwnStringController extends BaseController<String> {
}

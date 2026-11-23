package com.example.jdkidentity.app;

import com.example.jdkidentity.lib.BaseController;
import java.util.Date;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/jdk-dates")
public class DateController extends BaseController<Date> {
}

package com.example.jdkidentity.demand;

import com.example.jdkidentity.lib.BaseController;
import java.util.*;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/jdk-demand-dates")
public class DemandItemController extends BaseController<Date> {
}

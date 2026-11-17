package com.example.constant;

import static com.example.constant.shared.Regions.*;
import static com.example.constant.shared.Tenants.TENANT;

import com.example.constant.shared.Headers;
import com.example.constant.shared.Params;
import org.springframework.web.bind.annotation.CookieValue;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestHeader;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/constant-names")
public class ConstantNameController {

    static final String TRACE = "X-Trace-Id";

    @GetMapping("/{orderNo}")
    public void find(@RequestHeader(TRACE) String trace,
                     @RequestHeader(Headers.VERSION) String version,
                     @RequestHeader(value = Headers.LANGUAGE, defaultValue = "en") String language,
                     @RequestParam(Params.PAGE_SIZE) int size,
                     @PathVariable(name = Params.ORDER_NO) String no,
                     @CookieValue(Params.SESSION) String session,
                     @RequestParam(com.example.constant.shared.Params.KEYWORD) String keyword,
                     @RequestParam(TENANT) String tenant,
                     @RequestParam(REGION) String region,
                     @RequestHeader(Missing.NAME) String fallback,
                     @RequestParam("literal") String literal) {
    }
}

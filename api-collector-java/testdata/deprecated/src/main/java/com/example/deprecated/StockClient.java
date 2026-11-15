package com.example.deprecated;

import org.springframework.cloud.openfeign.FeignClient;
import org.springframework.web.bind.annotation.GetMapping;

@Deprecated
@FeignClient(name = "stock", path = "/stock")
public interface StockClient {

    /**
     * Stock level.
     */
    @GetMapping("/level")
    String level();
}

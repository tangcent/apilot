package com.example.demo.client;

import com.example.demo.model.DocumentedResponse;
import feign.Param;
import feign.RequestLine;
import org.springframework.cloud.openfeign.FeignClient;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestParam;

/**
 * Documented Feign client (Spring Cloud OpenFeign style)
 */
@FeignClient(name = "documented-service")
public interface DocumentedUserClient {

    /**
     * Fetch documented item.
     *
     * @param id documented item id
     * @param status fallback status
     */
    @GetMapping("/api/docs/{id}")
    DocumentedResponse getDocumented(@PathVariable Long id,
                                     @RequestParam(defaultValue = "active") String status);
}

/**
 * Documented Feign client (Netflix @RequestLine style)
 */
interface DocumentedLegacyClient {

    /**
     * Search documented items.
     *
     * @param keyword search keyword
     */
    @RequestLine("GET /api/docs/search?keyword={keyword}")
    DocumentedResponse searchDocumented(@Param("keyword") String keyword);
}

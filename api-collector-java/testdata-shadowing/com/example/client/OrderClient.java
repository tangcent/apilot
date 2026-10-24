package com.example.client;

import feign.Param;
import feign.RequestLine;

import com.example.dto.*;

import java.math.BigDecimal;

/**
 * Feign client that imports com.example.dto on demand and at the same time
 * declares its own RefundVO. The inline RefundVO is the one the handlers
 * return: a simple name declared in the same compilation unit shadows an
 * import-on-demand (JLS 6.4.1).
 */
public interface OrderClient {

    @RequestLine("GET /api/refunds/{id}")
    RefundVO getRefundInfo(@Param("id") Long id);

    @RequestLine("POST /api/refunds")
    RefundVO requestRefund(RefundReq req);
}

class RefundVO {

    private Long id;

    private Long orderId;

    private BigDecimal amount;

    private String reason;

    private Integer status;

    private String processedAt;
}

class RefundReq {

    private Long orderId;

    private BigDecimal amount;

    private String reason;
}

package com.example.dto;

import java.math.BigDecimal;

/**
 * Refund DTO held by the dto package. Several handlers in the project import
 * this package on demand while declaring their own RefundVO, which makes the
 * simple name ambiguous on purpose.
 */
public class RefundVO {

    private Long id;

    private String refundNo;

    private Long paymentId;

    private BigDecimal amount;
}

package com.example.dto;

import java.math.BigDecimal;

/**
 * Order statistics DTO held by the dto package. It carries one field more than
 * the copy declared inside AnalyticsController, so resolving the wrong one
 * shows up as an extra field in the export.
 */
public class OrderStatisticsVO {

    private Long totalOrders;

    private BigDecimal totalAmount;

    private Long pendingOrders;

    private Long completedOrders;

    private Long cancelledOrders;

    private BigDecimal averageOrderValue;
}

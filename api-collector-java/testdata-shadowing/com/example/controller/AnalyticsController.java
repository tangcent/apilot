package com.example.controller;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

import com.example.dto.*;

import java.math.BigDecimal;

/**
 * Handler that declares its own OrderStatisticsVO while importing
 * com.example.dto on demand. The inline declaration wins.
 */
@RestController
@RequestMapping("/api/analytics")
public class AnalyticsController {

    @GetMapping("/orders/statistics")
    public OrderStatisticsVO getOrderStatistics() {
        return new OrderStatisticsVO();
    }
}

class OrderStatisticsVO {

    private Long totalOrders;

    private BigDecimal totalAmount;

    private Long pendingOrders;

    private Long completedOrders;

    private Long cancelledOrders;
}

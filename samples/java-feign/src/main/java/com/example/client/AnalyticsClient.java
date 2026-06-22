package com.example.client;

import feign.Param;
import feign.RequestLine;
import com.example.model.Result;
import java.math.BigDecimal;
import java.util.List;
import java.util.Map;

public interface AnalyticsClient extends GenericBaseClient<AnalyticsOverviewVO> {

    @RequestLine("GET /api/analytics/overview")
    Result<AnalyticsOverviewVO> getOverview();

    @RequestLine("GET /api/analytics/sales?startDate={startDate}&endDate={endDate}&granularity={granularity}")
    Result<SalesStatisticsVO> getSalesStatistics(@Param("startDate") String startDate,
                                                   @Param("endDate") String endDate,
                                                   @Param("granularity") String granularity);

    @RequestLine("GET /api/analytics/users?startDate={startDate}&endDate={endDate}")
    Result<UserStatisticsVO> getUserStatistics(@Param("startDate") String startDate, @Param("endDate") String endDate);

    @RequestLine("GET /api/analytics/products?categoryId={categoryId}&limit={limit}")
    Result<ProductAnalyticsVO> getProductStatistics(@Param("categoryId") Long categoryId, @Param("limit") Integer limit);

    @RequestLine("GET /api/analytics/orders?startDate={startDate}&endDate={endDate}")
    Result<OrderAnalyticsVO> getOrderStatistics(@Param("startDate") String startDate, @Param("endDate") String endDate);

    @RequestLine("GET /api/analytics/trends?granularity={granularity}&limit={limit}")
    Result<List<TrendVO>> getTrends(@Param("granularity") String granularity, @Param("limit") Integer limit);

    @RequestLine("GET /api/analytics/dashboard")
    Result<Map<String, Object>> getDashboard();

    @RequestLine("GET /api/analytics/export?type={type}&startDate={startDate}&endDate={endDate}")
    Result<String> exportReport(@Param("type") String type, @Param("startDate") String startDate, @Param("endDate") String endDate);
}

class AnalyticsOverviewVO {
    private BigDecimal totalRevenue;
    private Long totalOrders;
    private Long totalUsers;
    private Long totalProducts;
    private BigDecimal todayRevenue;
    private Long todayOrders;
    private Long newUsers;
    private Double conversionRate;

    public BigDecimal getTotalRevenue() { return totalRevenue; }
    public void setTotalRevenue(BigDecimal totalRevenue) { this.totalRevenue = totalRevenue; }
    public Long getTotalOrders() { return totalOrders; }
    public void setTotalOrders(Long totalOrders) { this.totalOrders = totalOrders; }
    public Long getTotalUsers() { return totalUsers; }
    public void setTotalUsers(Long totalUsers) { this.totalUsers = totalUsers; }
    public Long getTotalProducts() { return totalProducts; }
    public void setTotalProducts(Long totalProducts) { this.totalProducts = totalProducts; }
    public BigDecimal getTodayRevenue() { return todayRevenue; }
    public void setTodayRevenue(BigDecimal todayRevenue) { this.todayRevenue = todayRevenue; }
    public Long getTodayOrders() { return todayOrders; }
    public void setTodayOrders(Long todayOrders) { this.todayOrders = todayOrders; }
    public Long getNewUsers() { return newUsers; }
    public void setNewUsers(Long newUsers) { this.newUsers = newUsers; }
    public Double getConversionRate() { return conversionRate; }
    public void setConversionRate(Double conversionRate) { this.conversionRate = conversionRate; }
}

class SalesStatisticsVO {
    private BigDecimal totalSales;
    private BigDecimal totalRefund;
    private Long totalOrders;
    private BigDecimal averageOrderValue;
    private List<DailySalesVO> dailySales;

    public BigDecimal getTotalSales() { return totalSales; }
    public void setTotalSales(BigDecimal totalSales) { this.totalSales = totalSales; }
    public BigDecimal getTotalRefund() { return totalRefund; }
    public void setTotalRefund(BigDecimal totalRefund) { this.totalRefund = totalRefund; }
    public Long getTotalOrders() { return totalOrders; }
    public void setTotalOrders(Long totalOrders) { this.totalOrders = totalOrders; }
    public BigDecimal getAverageOrderValue() { return averageOrderValue; }
    public void setAverageOrderValue(BigDecimal averageOrderValue) { this.averageOrderValue = averageOrderValue; }
    public List<DailySalesVO> getDailySales() { return dailySales; }
    public void setDailySales(List<DailySalesVO> dailySales) { this.dailySales = dailySales; }
}

class DailySalesVO {
    private String date;
    private BigDecimal sales;
    private Long orders;

    public String getDate() { return date; }
    public void setDate(String date) { this.date = date; }
    public BigDecimal getSales() { return sales; }
    public void setSales(BigDecimal sales) { this.sales = sales; }
    public Long getOrders() { return orders; }
    public void setOrders(Long orders) { this.orders = orders; }
}

class UserStatisticsVO {
    private Long totalUsers;
    private Long newUsers;
    private Long activeUsers;
    private Long payingUsers;
    private Double retentionRate;

    public Long getTotalUsers() { return totalUsers; }
    public void setTotalUsers(Long totalUsers) { this.totalUsers = totalUsers; }
    public Long getNewUsers() { return newUsers; }
    public void setNewUsers(Long newUsers) { this.newUsers = newUsers; }
    public Long getActiveUsers() { return activeUsers; }
    public void setActiveUsers(Long activeUsers) { this.activeUsers = activeUsers; }
    public Long getPayingUsers() { return payingUsers; }
    public void setPayingUsers(Long payingUsers) { this.payingUsers = payingUsers; }
    public Double getRetentionRate() { return retentionRate; }
    public void setRetentionRate(Double retentionRate) { this.retentionRate = retentionRate; }
}

class ProductAnalyticsVO {
    private Long totalProducts;
    private Long activeProducts;
    private Long outOfStockProducts;
    private List<TopProductVO> topProducts;

    public Long getTotalProducts() { return totalProducts; }
    public void setTotalProducts(Long totalProducts) { this.totalProducts = totalProducts; }
    public Long getActiveProducts() { return activeProducts; }
    public void setActiveProducts(Long activeProducts) { this.activeProducts = activeProducts; }
    public Long getOutOfStockProducts() { return outOfStockProducts; }
    public void setOutOfStockProducts(Long outOfStockProducts) { this.outOfStockProducts = outOfStockProducts; }
    public List<TopProductVO> getTopProducts() { return topProducts; }
    public void setTopProducts(List<TopProductVO> topProducts) { this.topProducts = topProducts; }
}

class TopProductVO {
    private Long productId;
    private String productName;
    private Long salesCount;
    private BigDecimal revenue;

    public Long getProductId() { return productId; }
    public void setProductId(Long productId) { this.productId = productId; }
    public String getProductName() { return productName; }
    public void setProductName(String productName) { this.productName = productName; }
    public Long getSalesCount() { return salesCount; }
    public void setSalesCount(Long salesCount) { this.salesCount = salesCount; }
    public BigDecimal getRevenue() { return revenue; }
    public void setRevenue(BigDecimal revenue) { this.revenue = revenue; }
}

class OrderAnalyticsVO {
    private Long totalOrders;
    private Long pendingOrders;
    private Long completedOrders;
    private Long cancelledOrders;
    private BigDecimal completionRate;

    public Long getTotalOrders() { return totalOrders; }
    public void setTotalOrders(Long totalOrders) { this.totalOrders = totalOrders; }
    public Long getPendingOrders() { return pendingOrders; }
    public void setPendingOrders(Long pendingOrders) { this.pendingOrders = pendingOrders; }
    public Long getCompletedOrders() { return completedOrders; }
    public void setCompletedOrders(Long completedOrders) { this.completedOrders = completedOrders; }
    public Long getCancelledOrders() { return cancelledOrders; }
    public void setCancelledOrders(Long cancelledOrders) { this.cancelledOrders = cancelledOrders; }
    public BigDecimal getCompletionRate() { return completionRate; }
    public void setCompletionRate(BigDecimal completionRate) { this.completionRate = completionRate; }
}

class TrendVO {
    private String date;
    private String metric;
    private BigDecimal value;

    public String getDate() { return date; }
    public void setDate(String date) { this.date = date; }
    public String getMetric() { return metric; }
    public void setMetric(String metric) { this.metric = metric; }
    public BigDecimal getValue() { return value; }
    public void setValue(BigDecimal value) { this.value = value; }
}

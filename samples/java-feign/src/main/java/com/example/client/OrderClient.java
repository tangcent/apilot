package com.example.client;

import feign.Param;
import feign.RequestLine;
import com.example.model.Order;
import com.example.model.Result;
import com.example.model.PageResult;
import java.math.BigDecimal;
import java.util.List;

public interface OrderClient extends BaseCrudClient<CreateOrderReq, OrderVO> {

    @RequestLine("POST /api/orders/{id}/cancel?reason={reason}")
    Result<Order> cancelOrder(@Param("id") Long id, @Param("reason") String reason);

    @RequestLine("GET /api/orders/{id}/track")
    Result<OrderTrackVO> trackOrder(@Param("id") Long id);

    @RequestLine("POST /api/orders/{id}/items")
    Result<Order> addOrderItem(@Param("id") Long id, AddOrderItemReq req);

    @RequestLine("POST /api/orders/{id}/refund")
    Result<RefundVO> requestRefund(@Param("id") Long id, RefundReq req);

    @RequestLine("GET /api/orders/statistics?startDate={startDate}&endDate={endDate}")
    Result<OrderStatisticsVO> getStatistics(@Param("startDate") String startDate, @Param("endDate") String endDate);

    @RequestLine("GET /api/orders/user/{userId}?status={status}&pageNum={pageNum}&pageSize={pageSize}")
    Result<PageResult<Order>> getOrdersByUser(@Param("userId") Long userId,
                                               @Param("status") Integer status,
                                               @Param("pageNum") Integer pageNum,
                                               @Param("pageSize") Integer pageSize);
}

class CreateOrderReq {
    private Long userId;
    private List<OrderItemReq> items;
    private String shippingAddress;
    private String receiverName;
    private String receiverPhone;
    private String remark;

    public Long getUserId() { return userId; }
    public void setUserId(Long userId) { this.userId = userId; }
    public List<OrderItemReq> getItems() { return items; }
    public void setItems(List<OrderItemReq> items) { this.items = items; }
    public String getShippingAddress() { return shippingAddress; }
    public void setShippingAddress(String shippingAddress) { this.shippingAddress = shippingAddress; }
    public String getReceiverName() { return receiverName; }
    public void setReceiverName(String receiverName) { this.receiverName = receiverName; }
    public String getReceiverPhone() { return receiverPhone; }
    public void setReceiverPhone(String receiverPhone) { this.receiverPhone = receiverPhone; }
    public String getRemark() { return remark; }
    public void setRemark(String remark) { this.remark = remark; }
}

class OrderItemReq {
    private Long productId;
    private Integer quantity;
    private BigDecimal price;

    public Long getProductId() { return productId; }
    public void setProductId(Long productId) { this.productId = productId; }
    public Integer getQuantity() { return quantity; }
    public void setQuantity(Integer quantity) { this.quantity = quantity; }
    public BigDecimal getPrice() { return price; }
    public void setPrice(BigDecimal price) { this.price = price; }
}

class OrderVO {
    private Long id;
    private String orderNo;
    private Long userId;
    private BigDecimal totalAmount;
    private BigDecimal finalAmount;
    private Integer status;
    private List<OrderItemVO> items;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getOrderNo() { return orderNo; }
    public void setOrderNo(String orderNo) { this.orderNo = orderNo; }
    public Long getUserId() { return userId; }
    public void setUserId(Long userId) { this.userId = userId; }
    public BigDecimal getTotalAmount() { return totalAmount; }
    public void setTotalAmount(BigDecimal totalAmount) { this.totalAmount = totalAmount; }
    public BigDecimal getFinalAmount() { return finalAmount; }
    public void setFinalAmount(BigDecimal finalAmount) { this.finalAmount = finalAmount; }
    public Integer getStatus() { return status; }
    public void setStatus(Integer status) { this.status = status; }
    public List<OrderItemVO> getItems() { return items; }
    public void setItems(List<OrderItemVO> items) { this.items = items; }
}

class OrderItemVO {
    private Long productId;
    private String productName;
    private Integer quantity;
    private BigDecimal price;

    public Long getProductId() { return productId; }
    public void setProductId(Long productId) { this.productId = productId; }
    public String getProductName() { return productName; }
    public void setProductName(String productName) { this.productName = productName; }
    public Integer getQuantity() { return quantity; }
    public void setQuantity(Integer quantity) { this.quantity = quantity; }
    public BigDecimal getPrice() { return price; }
    public void setPrice(BigDecimal price) { this.price = price; }
}

class OrderTrackVO {
    private String orderNo;
    private Integer status;
    private List<TrackPointVO> trackPoints;

    public String getOrderNo() { return orderNo; }
    public void setOrderNo(String orderNo) { this.orderNo = orderNo; }
    public Integer getStatus() { return status; }
    public void setStatus(Integer status) { this.status = status; }
    public List<TrackPointVO> getTrackPoints() { return trackPoints; }
    public void setTrackPoints(List<TrackPointVO> trackPoints) { this.trackPoints = trackPoints; }
}

class TrackPointVO {
    private String time;
    private String description;

    public String getTime() { return time; }
    public void setTime(String time) { this.time = time; }
    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }
}

class AddOrderItemReq {
    private Long productId;
    private Integer quantity;
    private BigDecimal price;

    public Long getProductId() { return productId; }
    public void setProductId(Long productId) { this.productId = productId; }
    public Integer getQuantity() { return quantity; }
    public void setQuantity(Integer quantity) { this.quantity = quantity; }
    public BigDecimal getPrice() { return price; }
    public void setPrice(BigDecimal price) { this.price = price; }
}

class RefundReq {
    private String reason;
    private BigDecimal amount;

    public String getReason() { return reason; }
    public void setReason(String reason) { this.reason = reason; }
    public BigDecimal getAmount() { return amount; }
    public void setAmount(BigDecimal amount) { this.amount = amount; }
}

class RefundVO {
    private Long id;
    private String refundNo;
    private BigDecimal amount;
    private Integer status;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getRefundNo() { return refundNo; }
    public void setRefundNo(String refundNo) { this.refundNo = refundNo; }
    public BigDecimal getAmount() { return amount; }
    public void setAmount(BigDecimal amount) { this.amount = amount; }
    public Integer getStatus() { return status; }
    public void setStatus(Integer status) { this.status = status; }
}

class OrderStatisticsVO {
    private Long totalOrders;
    private BigDecimal totalAmount;
    private Long pendingOrders;
    private Long completedOrders;
    private Long cancelledOrders;

    public Long getTotalOrders() { return totalOrders; }
    public void setTotalOrders(Long totalOrders) { this.totalOrders = totalOrders; }
    public BigDecimal getTotalAmount() { return totalAmount; }
    public void setTotalAmount(BigDecimal totalAmount) { this.totalAmount = totalAmount; }
    public Long getPendingOrders() { return pendingOrders; }
    public void setPendingOrders(Long pendingOrders) { this.pendingOrders = pendingOrders; }
    public Long getCompletedOrders() { return completedOrders; }
    public void setCompletedOrders(Long completedOrders) { this.completedOrders = completedOrders; }
    public Long getCancelledOrders() { return cancelledOrders; }
    public void setCancelledOrders(Long cancelledOrders) { this.cancelledOrders = cancelledOrders; }
}

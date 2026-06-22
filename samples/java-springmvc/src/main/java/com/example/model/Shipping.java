package com.example.model;

import java.time.LocalDateTime;

public class Shipping extends BaseEntity {
    private String shippingNo;
    private Long orderId;
    private String carrier;
    private String trackingNo;
    private Integer status;
    private LocalDateTime shippedAt;
    private LocalDateTime deliveredAt;

    public String getShippingNo() { return shippingNo; }
    public void setShippingNo(String shippingNo) { this.shippingNo = shippingNo; }
    public Long getOrderId() { return orderId; }
    public void setOrderId(Long orderId) { this.orderId = orderId; }
    public String getCarrier() { return carrier; }
    public void setCarrier(String carrier) { this.carrier = carrier; }
    public String getTrackingNo() { return trackingNo; }
    public void setTrackingNo(String trackingNo) { this.trackingNo = trackingNo; }
    public Integer getStatus() { return status; }
    public void setStatus(Integer status) { this.status = status; }
    public LocalDateTime getShippedAt() { return shippedAt; }
    public void setShippedAt(LocalDateTime shippedAt) { this.shippedAt = shippedAt; }
    public LocalDateTime getDeliveredAt() { return deliveredAt; }
    public void setDeliveredAt(LocalDateTime deliveredAt) { this.deliveredAt = deliveredAt; }
}

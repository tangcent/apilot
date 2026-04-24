package com.example.demo.model;

import java.util.List;
import java.util.Map;

public class OrderVO {
    private Long id;
    private String orderId;
    private String customerName;
    private double total;
    private List<String> tags;
    private Map<String, Object> attributes;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getOrderId() { return orderId; }
    public void setOrderId(String orderId) { this.orderId = orderId; }
    public String getCustomerName() { return customerName; }
    public void setCustomerName(String customerName) { this.customerName = customerName; }
    public double getTotal() { return total; }
    public void setTotal(double total) { this.total = total; }
    public List<String> getTags() { return tags; }
    public void setTags(List<String> tags) { this.tags = tags; }
    public Map<String, Object> getAttributes() { return attributes; }
    public void setAttributes(Map<String, Object> attributes) { this.attributes = attributes; }
}

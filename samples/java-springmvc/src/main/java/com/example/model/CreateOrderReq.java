package com.example.model;

import java.util.List;
import java.util.Map;

public class CreateOrderReq {
    private String orderId;
    private String customerName;
    private double amount;
    private List<String> items;
    private Map<String, String> metadata;

    public String getOrderId() { return orderId; }
    public void setOrderId(String orderId) { this.orderId = orderId; }
    public String getCustomerName() { return customerName; }
    public void setCustomerName(String customerName) { this.customerName = customerName; }
    public double getAmount() { return amount; }
    public void setAmount(double amount) { this.amount = amount; }
    public List<String> getItems() { return items; }
    public void setItems(List<String> items) { this.items = items; }
    public Map<String, String> getMetadata() { return metadata; }
    public void setMetadata(Map<String, String> metadata) { this.metadata = metadata; }
}

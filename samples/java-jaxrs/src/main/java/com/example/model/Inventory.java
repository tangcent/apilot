package com.example.model;

public class Inventory extends BaseEntity {
    private Long productId;
    private Long warehouseId;
    private Integer quantity;
    private Integer locked;
    private Integer available;

    public Long getProductId() { return productId; }
    public void setProductId(Long productId) { this.productId = productId; }
    public Long getWarehouseId() { return warehouseId; }
    public void setWarehouseId(Long warehouseId) { this.warehouseId = warehouseId; }
    public Integer getQuantity() { return quantity; }
    public void setQuantity(Integer quantity) { this.quantity = quantity; }
    public Integer getLocked() { return locked; }
    public void setLocked(Integer locked) { this.locked = locked; }
    public Integer getAvailable() { return available; }
    public void setAvailable(Integer available) { this.available = available; }
}

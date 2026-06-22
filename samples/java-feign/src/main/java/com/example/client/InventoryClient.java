package com.example.client;

import feign.Param;
import feign.RequestLine;
import com.example.model.Inventory;
import com.example.model.Result;
import java.util.List;

public interface InventoryClient extends BaseCrudClient<CreateInventoryReq, Inventory> {

    @RequestLine("PUT /api/inventory/{id}/stock?quantity={quantity}")
    Result<Inventory> updateStock(@Param("id") Long id, @Param("quantity") Integer quantity);

    @RequestLine("GET /api/inventory/{id}/movements?pageNum={pageNum}&pageSize={pageSize}")
    Result<List<InventoryMovementVO>> getMovements(@Param("id") Long id,
                                                    @Param("pageNum") Integer pageNum,
                                                    @Param("pageSize") Integer pageSize);

    @RequestLine("GET /api/inventory/alerts?threshold={threshold}")
    Result<List<InventoryAlertVO>> getAlerts(@Param("threshold") Integer threshold);

    @RequestLine("GET /api/inventory/warehouses")
    Result<List<WarehouseVO>> getWarehouses();

    @RequestLine("POST /api/inventory/batch-adjust")
    Result<Void> batchAdjust(List<AdjustStockReq> reqs);

    @RequestLine("GET /api/inventory/product/{productId}")
    Result<List<Inventory>> getByProduct(@Param("productId") Long productId);
}

class CreateInventoryReq {
    private Long productId;
    private Long warehouseId;
    private Integer quantity;

    public Long getProductId() { return productId; }
    public void setProductId(Long productId) { this.productId = productId; }
    public Long getWarehouseId() { return warehouseId; }
    public void setWarehouseId(Long warehouseId) { this.warehouseId = warehouseId; }
    public Integer getQuantity() { return quantity; }
    public void setQuantity(Integer quantity) { this.quantity = quantity; }
}

class InventoryMovementVO {
    private Long id;
    private Long productId;
    private Integer quantity;
    private String type;
    private String reason;
    private String createdAt;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public Long getProductId() { return productId; }
    public void setProductId(Long productId) { this.productId = productId; }
    public Integer getQuantity() { return quantity; }
    public void setQuantity(Integer quantity) { this.quantity = quantity; }
    public String getType() { return type; }
    public void setType(String type) { this.type = type; }
    public String getReason() { return reason; }
    public void setReason(String reason) { this.reason = reason; }
    public String getCreatedAt() { return createdAt; }
    public void setCreatedAt(String createdAt) { this.createdAt = createdAt; }
}

class InventoryAlertVO {
    private Long productId;
    private String productName;
    private Integer currentStock;
    private Integer threshold;
    private String warehouseName;

    public Long getProductId() { return productId; }
    public void setProductId(Long productId) { this.productId = productId; }
    public String getProductName() { return productName; }
    public void setProductName(String productName) { this.productName = productName; }
    public Integer getCurrentStock() { return currentStock; }
    public void setCurrentStock(Integer currentStock) { this.currentStock = currentStock; }
    public Integer getThreshold() { return threshold; }
    public void setThreshold(Integer threshold) { this.threshold = threshold; }
    public String getWarehouseName() { return warehouseName; }
    public void setWarehouseName(String warehouseName) { this.warehouseName = warehouseName; }
}

class WarehouseVO {
    private Long id;
    private String name;
    private String address;
    private String contact;
    private String phone;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getAddress() { return address; }
    public void setAddress(String address) { this.address = address; }
    public String getContact() { return contact; }
    public void setContact(String contact) { this.contact = contact; }
    public String getPhone() { return phone; }
    public void setPhone(String phone) { this.phone = phone; }
}

class AdjustStockReq {
    private Long productId;
    private Long warehouseId;
    private Integer quantity;
    private String reason;

    public Long getProductId() { return productId; }
    public void setProductId(Long productId) { this.productId = productId; }
    public Long getWarehouseId() { return warehouseId; }
    public void setWarehouseId(Long warehouseId) { this.warehouseId = warehouseId; }
    public Integer getQuantity() { return quantity; }
    public void setQuantity(Integer quantity) { this.quantity = quantity; }
    public String getReason() { return reason; }
    public void setReason(String reason) { this.reason = reason; }
}

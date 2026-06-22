package com.example.controller;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import com.example.model.Inventory;
import com.example.model.Result;
import java.util.List;

@Path("/api/inventory")
public class InventoryController extends BaseCrudController<CreateInventoryReq, Inventory> {

    @PUT
    @Path("/{id}/stock")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<Inventory> updateStock(@PathParam("id") Long id, @QueryParam("quantity") Integer quantity) {
        return Result.success(new Inventory());
    }

    @GET
    @Path("/{id}/movements")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<InventoryMovementVO>> getMovements(
            @PathParam("id") Long id,
            @DefaultValue("1") @QueryParam("pageNum") Integer pageNum,
            @DefaultValue("10") @QueryParam("pageSize") Integer pageSize) {
        return Result.success(List.of());
    }

    @GET
    @Path("/alerts")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<InventoryAlertVO>> getAlerts(@DefaultValue("10") @QueryParam("threshold") Integer threshold) {
        return Result.success(List.of());
    }

    @GET
    @Path("/warehouses")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<WarehouseVO>> getWarehouses() {
        return Result.success(List.of());
    }

    @POST
    @Path("/batch-adjust")
    @Consumes(MediaType.APPLICATION_JSON)
    @Produces(MediaType.APPLICATION_JSON)
    public Result<Void> batchAdjust(List<AdjustStockReq> reqs) {
        return Result.success(null);
    }

    @GET
    @Path("/product/{productId}")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<Inventory>> getByProduct(@PathParam("productId") Long productId) {
        return Result.success(List.of());
    }
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

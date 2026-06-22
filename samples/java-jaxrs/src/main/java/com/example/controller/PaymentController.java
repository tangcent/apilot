package com.example.controller;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import com.example.model.Payment;
import com.example.model.Result;
import java.math.BigDecimal;
import java.util.List;

@Path("/api/payments")
public class PaymentController extends BaseCrudController<CreatePaymentReq, Payment> {

    @POST
    @Path("/{id}/refund")
    @Consumes(MediaType.APPLICATION_JSON)
    @Produces(MediaType.APPLICATION_JSON)
    public Result<PaymentRefundVO> refundPayment(@PathParam("id") Long id, PaymentRefundReq req) {
        return Result.success(new PaymentRefundVO());
    }

    @GET
    @Path("/{id}/records")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<PaymentRecordVO>> getPaymentRecords(@PathParam("id") Long id) {
        return Result.success(List.of());
    }

    @GET
    @Path("/channels")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<PaymentChannelVO>> getPaymentChannels() {
        return Result.success(List.of());
    }

    @GET
    @Path("/statistics")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<PaymentStatisticsVO> getStatistics(
            @QueryParam("startDate") String startDate,
            @QueryParam("endDate") String endDate) {
        return Result.success(new PaymentStatisticsVO());
    }

    @POST
    @Path("/{id}/confirm")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<Payment> confirmPayment(@PathParam("id") Long id) {
        return Result.success(new Payment());
    }
}

class CreatePaymentReq {
    private Long orderId;
    private BigDecimal amount;
    private String method;

    public Long getOrderId() { return orderId; }
    public void setOrderId(Long orderId) { this.orderId = orderId; }
    public BigDecimal getAmount() { return amount; }
    public void setAmount(BigDecimal amount) { this.amount = amount; }
    public String getMethod() { return method; }
    public void setMethod(String method) { this.method = method; }
}

class PaymentRefundReq {
    private String reason;
    private BigDecimal amount;

    public String getReason() { return reason; }
    public void setReason(String reason) { this.reason = reason; }
    public BigDecimal getAmount() { return amount; }
    public void setAmount(BigDecimal amount) { this.amount = amount; }
}

class PaymentRefundVO {
    private Long id;
    private String refundNo;
    private BigDecimal amount;
    private Integer status;
    private String createdAt;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getRefundNo() { return refundNo; }
    public void setRefundNo(String refundNo) { this.refundNo = refundNo; }
    public BigDecimal getAmount() { return amount; }
    public void setAmount(BigDecimal amount) { this.amount = amount; }
    public Integer getStatus() { return status; }
    public void setStatus(Integer status) { this.status = status; }
    public String getCreatedAt() { return createdAt; }
    public void setCreatedAt(String createdAt) { this.createdAt = createdAt; }
}

class PaymentRecordVO {
    private Long id;
    private String action;
    private BigDecimal amount;
    private Integer status;
    private String createdAt;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getAction() { return action; }
    public void setAction(String action) { this.action = action; }
    public BigDecimal getAmount() { return amount; }
    public void setAmount(BigDecimal amount) { this.amount = amount; }
    public Integer getStatus() { return status; }
    public void setStatus(Integer status) { this.status = status; }
    public String getCreatedAt() { return createdAt; }
    public void setCreatedAt(String createdAt) { this.createdAt = createdAt; }
}

class PaymentChannelVO {
    private String code;
    private String name;
    private String icon;
    private Boolean enabled;

    public String getCode() { return code; }
    public void setCode(String code) { this.code = code; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getIcon() { return icon; }
    public void setIcon(String icon) { this.icon = icon; }
    public Boolean getEnabled() { return enabled; }
    public void setEnabled(Boolean enabled) { this.enabled = enabled; }
}

class PaymentStatisticsVO {
    private BigDecimal totalAmount;
    private BigDecimal refundAmount;
    private Long totalPayments;
    private Long refundCount;

    public BigDecimal getTotalAmount() { return totalAmount; }
    public void setTotalAmount(BigDecimal totalAmount) { this.totalAmount = totalAmount; }
    public BigDecimal getRefundAmount() { return refundAmount; }
    public void setRefundAmount(BigDecimal refundAmount) { this.refundAmount = refundAmount; }
    public Long getTotalPayments() { return totalPayments; }
    public void setTotalPayments(Long totalPayments) { this.totalPayments = totalPayments; }
    public Long getRefundCount() { return refundCount; }
    public void setRefundCount(Long refundCount) { this.refundCount = refundCount; }
}

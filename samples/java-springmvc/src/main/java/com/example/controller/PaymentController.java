package com.example.controller;

import org.springframework.web.bind.annotation.*;
import com.example.model.Payment;
import com.example.model.Result;
import com.example.model.PageResult;
import java.math.BigDecimal;
import java.util.List;

@RestController
@RequestMapping("/api/payments")
public class PaymentController extends BaseCrudController<CreatePaymentReq, Payment> {

    @PostMapping("/{id}/refund")
    public Result<PaymentRefundVO> refundPayment(@PathVariable Long id, @RequestBody PaymentRefundReq req) {
        return Result.success(new PaymentRefundVO());
    }

    @GetMapping("/{id}/records")
    public Result<List<PaymentRecordVO>> getPaymentRecords(@PathVariable Long id) {
        return Result.success(List.of());
    }

    @GetMapping("/channels")
    public Result<List<PaymentChannelVO>> getPaymentChannels() {
        return Result.success(List.of());
    }

    @GetMapping("/statistics")
    public Result<PaymentStatisticsVO> getStatistics(
            @RequestParam(required = false) String startDate,
            @RequestParam(required = false) String endDate) {
        return Result.success(new PaymentStatisticsVO());
    }

    @PostMapping("/{id}/confirm")
    public Result<Payment> confirmPayment(@PathVariable Long id) {
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

package com.example.client;

import feign.Param;
import feign.RequestLine;
import com.example.model.Shipping;
import com.example.model.Result;
import java.math.BigDecimal;
import java.util.List;

public interface ShippingClient extends BaseCrudClient<CreateShippingReq, Shipping> {

    @RequestLine("GET /api/shipping/{id}/track")
    Result<ShippingTrackVO> trackShipment(@Param("id") Long id);

    @RequestLine("GET /api/shipping/carriers")
    Result<List<CarrierVO>> getCarriers();

    @RequestLine("POST /api/shipping/rates")
    Result<ShippingRateVO> calculateRates(ShippingRateReq req);

    @RequestLine("POST /api/shipping/{id}/ship?carrier={carrier}&trackingNo={trackingNo}")
    Result<Shipping> shipOrder(@Param("id") Long id, @Param("carrier") String carrier, @Param("trackingNo") String trackingNo);

    @RequestLine("POST /api/shipping/{id}/deliver")
    Result<Shipping> markDelivered(@Param("id") Long id);

    @RequestLine("GET /api/shipping/statistics?startDate={startDate}&endDate={endDate}")
    Result<ShippingStatisticsVO> getStatistics(@Param("startDate") String startDate, @Param("endDate") String endDate);
}

class CreateShippingReq {
    private Long orderId;
    private String carrier;
    private String trackingNo;

    public Long getOrderId() { return orderId; }
    public void setOrderId(Long orderId) { this.orderId = orderId; }
    public String getCarrier() { return carrier; }
    public void setCarrier(String carrier) { this.carrier = carrier; }
    public String getTrackingNo() { return trackingNo; }
    public void setTrackingNo(String trackingNo) { this.trackingNo = trackingNo; }
}

class ShippingTrackVO {
    private String shippingNo;
    private String carrier;
    private String trackingNo;
    private Integer status;
    private List<TrackPointVO> trackPoints;

    public String getShippingNo() { return shippingNo; }
    public void setShippingNo(String shippingNo) { this.shippingNo = shippingNo; }
    public String getCarrier() { return carrier; }
    public void setCarrier(String carrier) { this.carrier = carrier; }
    public String getTrackingNo() { return trackingNo; }
    public void setTrackingNo(String trackingNo) { this.trackingNo = trackingNo; }
    public Integer getStatus() { return status; }
    public void setStatus(Integer status) { this.status = status; }
    public List<TrackPointVO> getTrackPoints() { return trackPoints; }
    public void setTrackPoints(List<TrackPointVO> trackPoints) { this.trackPoints = trackPoints; }
}

class CarrierVO {
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

class ShippingRateReq {
    private String fromAddress;
    private String toAddress;
    private BigDecimal weight;
    private String carrier;

    public String getFromAddress() { return fromAddress; }
    public void setFromAddress(String fromAddress) { this.fromAddress = fromAddress; }
    public String getToAddress() { return toAddress; }
    public void setToAddress(String toAddress) { this.toAddress = toAddress; }
    public BigDecimal getWeight() { return weight; }
    public void setWeight(BigDecimal weight) { this.weight = weight; }
    public String getCarrier() { return carrier; }
    public void setCarrier(String carrier) { this.carrier = carrier; }
}

class ShippingRateVO {
    private String carrier;
    private BigDecimal fee;
    private Integer estimatedDays;
    private String description;

    public String getCarrier() { return carrier; }
    public void setCarrier(String carrier) { this.carrier = carrier; }
    public BigDecimal getFee() { return fee; }
    public void setFee(BigDecimal fee) { this.fee = fee; }
    public Integer getEstimatedDays() { return estimatedDays; }
    public void setEstimatedDays(Integer estimatedDays) { this.estimatedDays = estimatedDays; }
    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }
}

class ShippingStatisticsVO {
    private Long totalShipments;
    private Long pendingShipments;
    private Long deliveredShipments;
    private BigDecimal averageDays;

    public Long getTotalShipments() { return totalShipments; }
    public void setTotalShipments(Long totalShipments) { this.totalShipments = totalShipments; }
    public Long getPendingShipments() { return pendingShipments; }
    public void setPendingShipments(Long pendingShipments) { this.pendingShipments = pendingShipments; }
    public Long getDeliveredShipments() { return deliveredShipments; }
    public void setDeliveredShipments(Long deliveredShipments) { this.deliveredShipments = deliveredShipments; }
    public BigDecimal getAverageDays() { return averageDays; }
    public void setAverageDays(BigDecimal averageDays) { this.averageDays = averageDays; }
}

package com.example.controller;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import com.example.model.Product;
import com.example.model.Result;
import com.example.model.PageResult;
import java.math.BigDecimal;
import java.util.List;

@Path("/api/products")
public class ProductController extends BaseCrudController<CreateProductReq, Product> {

    @GET
    @Path("/search")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<PageResult<Product>> searchProducts(
            @QueryParam("keyword") String keyword,
            @QueryParam("categoryId") Long categoryId,
            @QueryParam("minPrice") BigDecimal minPrice,
            @QueryParam("maxPrice") BigDecimal maxPrice,
            @DefaultValue("1") @QueryParam("pageNum") Integer pageNum,
            @DefaultValue("10") @QueryParam("pageSize") Integer pageSize) {
        return Result.success(new PageResult<>());
    }

    @GET
    @Path("/{id}/detail")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<ProductDetailVO> getProductDetail(@PathParam("id") Long id) {
        return Result.success(new ProductDetailVO());
    }

    @GET
    @Path("/category/{categoryId}")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<PageResult<Product>> getProductsByCategory(
            @PathParam("categoryId") Long categoryId,
            @DefaultValue("1") @QueryParam("pageNum") Integer pageNum,
            @DefaultValue("10") @QueryParam("pageSize") Integer pageSize) {
        return Result.success(new PageResult<>());
    }

    @GET
    @Path("/hot")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<Product>> getHotProducts(@DefaultValue("10") @QueryParam("limit") Integer limit) {
        return Result.success(List.of());
    }

    @GET
    @Path("/new")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<Product>> getNewProducts(@DefaultValue("10") @QueryParam("limit") Integer limit) {
        return Result.success(List.of());
    }

    @POST
    @Path("/{id}/specs")
    @Consumes(MediaType.APPLICATION_JSON)
    @Produces(MediaType.APPLICATION_JSON)
    public Result<ProductSpecVO> addSpec(@PathParam("id") Long id, CreateProductSpecReq req) {
        return Result.success(new ProductSpecVO());
    }

    @GET
    @Path("/{id}/specs")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<ProductSpecVO>> getSpecs(@PathParam("id") Long id) {
        return Result.success(List.of());
    }

    @GET
    @Path("/{id}/reviews")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<PageResult<Object>> getReviews(
            @PathParam("id") Long id,
            @DefaultValue("1") @QueryParam("pageNum") Integer pageNum,
            @DefaultValue("10") @QueryParam("pageSize") Integer pageSize) {
        return Result.success(new PageResult<>());
    }

    @GET
    @Path("/{id}/statistics")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<ProductStatisticsVO> getStatistics(@PathParam("id") Long id) {
        return Result.success(new ProductStatisticsVO());
    }

    @POST
    @Path("/batch")
    @Consumes(MediaType.APPLICATION_JSON)
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<Product>> batchCreate(List<CreateProductReq> reqs) {
        return Result.success(List.of());
    }
}

class CreateProductReq {
    private String name;
    private String description;
    private BigDecimal price;
    private BigDecimal originalPrice;
    private Long categoryId;
    private Integer stock;
    private String mainImage;
    private String unit;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }
    public BigDecimal getPrice() { return price; }
    public void setPrice(BigDecimal price) { this.price = price; }
    public BigDecimal getOriginalPrice() { return originalPrice; }
    public void setOriginalPrice(BigDecimal originalPrice) { this.originalPrice = originalPrice; }
    public Long getCategoryId() { return categoryId; }
    public void setCategoryId(Long categoryId) { this.categoryId = categoryId; }
    public Integer getStock() { return stock; }
    public void setStock(Integer stock) { this.stock = stock; }
    public String getMainImage() { return mainImage; }
    public void setMainImage(String mainImage) { this.mainImage = mainImage; }
    public String getUnit() { return unit; }
    public void setUnit(String unit) { this.unit = unit; }
}

class ProductDetailVO {
    private Long id;
    private String name;
    private String description;
    private BigDecimal price;
    private BigDecimal originalPrice;
    private Long categoryId;
    private String categoryName;
    private Integer stock;
    private Integer sales;
    private String mainImage;
    private List<String> images;
    private List<ProductSpecVO> specs;
    private Double averageRating;
    private Integer reviewCount;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getDescription() { return description; }
    public void setDescription(String description) { this.description = description; }
    public BigDecimal getPrice() { return price; }
    public void setPrice(BigDecimal price) { this.price = price; }
    public BigDecimal getOriginalPrice() { return originalPrice; }
    public void setOriginalPrice(BigDecimal originalPrice) { this.originalPrice = originalPrice; }
    public Long getCategoryId() { return categoryId; }
    public void setCategoryId(Long categoryId) { this.categoryId = categoryId; }
    public String getCategoryName() { return categoryName; }
    public void setCategoryName(String categoryName) { this.categoryName = categoryName; }
    public Integer getStock() { return stock; }
    public void setStock(Integer stock) { this.stock = stock; }
    public Integer getSales() { return sales; }
    public void setSales(Integer sales) { this.sales = sales; }
    public String getMainImage() { return mainImage; }
    public void setMainImage(String mainImage) { this.mainImage = mainImage; }
    public List<String> getImages() { return images; }
    public void setImages(List<String> images) { this.images = images; }
    public List<ProductSpecVO> getSpecs() { return specs; }
    public void setSpecs(List<ProductSpecVO> specs) { this.specs = specs; }
    public Double getAverageRating() { return averageRating; }
    public void setAverageRating(Double averageRating) { this.averageRating = averageRating; }
    public Integer getReviewCount() { return reviewCount; }
    public void setReviewCount(Integer reviewCount) { this.reviewCount = reviewCount; }
}

class ProductSpecVO {
    private Long id;
    private String name;
    private String value;
    private BigDecimal price;
    private Integer stock;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getValue() { return value; }
    public void setValue(String value) { this.value = value; }
    public BigDecimal getPrice() { return price; }
    public void setPrice(BigDecimal price) { this.price = price; }
    public Integer getStock() { return stock; }
    public void setStock(Integer stock) { this.stock = stock; }
}

class CreateProductSpecReq {
    private String name;
    private String value;
    private BigDecimal price;
    private Integer stock;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public String getValue() { return value; }
    public void setValue(String value) { this.value = value; }
    public BigDecimal getPrice() { return price; }
    public void setPrice(BigDecimal price) { this.price = price; }
    public Integer getStock() { return stock; }
    public void setStock(Integer stock) { this.stock = stock; }
}

class ProductStatisticsVO {
    private Long productId;
    private Integer views;
    private Integer sales;
    private Integer favorites;
    private Double averageRating;
    private Integer reviewCount;

    public Long getProductId() { return productId; }
    public void setProductId(Long productId) { this.productId = productId; }
    public Integer getViews() { return views; }
    public void setViews(Integer views) { this.views = views; }
    public Integer getSales() { return sales; }
    public void setSales(Integer sales) { this.sales = sales; }
    public Integer getFavorites() { return favorites; }
    public void setFavorites(Integer favorites) { this.favorites = favorites; }
    public Double getAverageRating() { return averageRating; }
    public void setAverageRating(Double averageRating) { this.averageRating = averageRating; }
    public Integer getReviewCount() { return reviewCount; }
    public void setReviewCount(Integer reviewCount) { this.reviewCount = reviewCount; }
}

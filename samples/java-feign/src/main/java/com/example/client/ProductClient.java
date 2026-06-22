package com.example.client;

import feign.Param;
import feign.RequestLine;
import com.example.model.Product;
import com.example.model.Result;
import com.example.model.PageResult;
import java.math.BigDecimal;
import java.util.List;

public interface ProductClient extends BaseCrudClient<CreateProductReq, Product> {

    @RequestLine("GET /api/products/search?keyword={keyword}&categoryId={categoryId}&minPrice={minPrice}&maxPrice={maxPrice}&pageNum={pageNum}&pageSize={pageSize}")
    Result<PageResult<Product>> searchProducts(@Param("keyword") String keyword,
                                                @Param("categoryId") Long categoryId,
                                                @Param("minPrice") BigDecimal minPrice,
                                                @Param("maxPrice") BigDecimal maxPrice,
                                                @Param("pageNum") Integer pageNum,
                                                @Param("pageSize") Integer pageSize);

    @RequestLine("GET /api/products/{id}/detail")
    Result<ProductDetailVO> getProductDetail(@Param("id") Long id);

    @RequestLine("GET /api/products/category/{categoryId}?pageNum={pageNum}&pageSize={pageSize}")
    Result<PageResult<Product>> getProductsByCategory(@Param("categoryId") Long categoryId,
                                                       @Param("pageNum") Integer pageNum,
                                                       @Param("pageSize") Integer pageSize);

    @RequestLine("GET /api/products/hot?limit={limit}")
    Result<List<Product>> getHotProducts(@Param("limit") Integer limit);

    @RequestLine("GET /api/products/new?limit={limit}")
    Result<List<Product>> getNewProducts(@Param("limit") Integer limit);

    @RequestLine("POST /api/products/{id}/specs")
    Result<ProductSpecVO> addSpec(@Param("id") Long id, CreateProductSpecReq req);

    @RequestLine("GET /api/products/{id}/specs")
    Result<List<ProductSpecVO>> getSpecs(@Param("id") Long id);

    @RequestLine("GET /api/products/{id}/reviews?pageNum={pageNum}&pageSize={pageSize}")
    Result<PageResult<Object>> getReviews(@Param("id") Long id,
                                           @Param("pageNum") Integer pageNum,
                                           @Param("pageSize") Integer pageSize);

    @RequestLine("GET /api/products/{id}/statistics")
    Result<ProductStatisticsVO> getStatistics(@Param("id") Long id);

    @RequestLine("POST /api/products/batch")
    Result<List<Product>> batchCreate(List<CreateProductReq> reqs);
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

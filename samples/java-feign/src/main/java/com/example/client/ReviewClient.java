package com.example.client;

import feign.Param;
import feign.RequestLine;
import com.example.model.Review;
import com.example.model.Result;
import com.example.model.PageResult;
import java.util.List;

public interface ReviewClient extends BaseCrudClient<CreateReviewReq, Review> {

    @RequestLine("GET /api/reviews/product/{productId}?rating={rating}&pageNum={pageNum}&pageSize={pageSize}")
    Result<PageResult<Review>> getReviewsByProduct(@Param("productId") Long productId,
                                                     @Param("rating") Integer rating,
                                                     @Param("pageNum") Integer pageNum,
                                                     @Param("pageSize") Integer pageSize);

    @RequestLine("POST /api/reviews/{id}/comments")
    Result<CommentVO> addComment(@Param("id") Long id, CreateCommentReq req);

    @RequestLine("GET /api/reviews/{id}/comments")
    Result<List<CommentVO>> getComments(@Param("id") Long id);

    @RequestLine("PUT /api/reviews/{id}/moderate?status={status}")
    Result<Review> moderateReview(@Param("id") Long id, @Param("status") Integer status);

    @RequestLine("GET /api/reviews/user/{userId}?pageNum={pageNum}&pageSize={pageSize}")
    Result<PageResult<Review>> getReviewsByUser(@Param("userId") Long userId,
                                                 @Param("pageNum") Integer pageNum,
                                                 @Param("pageSize") Integer pageSize);

    @RequestLine("GET /api/reviews/statistics?productId={productId}")
    Result<ReviewStatisticsVO> getStatistics(@Param("productId") Long productId);
}

class CreateReviewReq {
    private Long productId;
    private Integer rating;
    private String content;
    private String images;

    public Long getProductId() { return productId; }
    public void setProductId(Long productId) { this.productId = productId; }
    public Integer getRating() { return rating; }
    public void setRating(Integer rating) { this.rating = rating; }
    public String getContent() { return content; }
    public void setContent(String content) { this.content = content; }
    public String getImages() { return images; }
    public void setImages(String images) { this.images = images; }
}

class CreateCommentReq {
    private String content;
    private Long parentId;

    public String getContent() { return content; }
    public void setContent(String content) { this.content = content; }
    public Long getParentId() { return parentId; }
    public void setParentId(Long parentId) { this.parentId = parentId; }
}

class CommentVO {
    private Long id;
    private Long reviewId;
    private Long userId;
    private String username;
    private String content;
    private String createdAt;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public Long getReviewId() { return reviewId; }
    public void setReviewId(Long reviewId) { this.reviewId = reviewId; }
    public Long getUserId() { return userId; }
    public void setUserId(Long userId) { this.userId = userId; }
    public String getUsername() { return username; }
    public void setUsername(String username) { this.username = username; }
    public String getContent() { return content; }
    public void setContent(String content) { this.content = content; }
    public String getCreatedAt() { return createdAt; }
    public void setCreatedAt(String createdAt) { this.createdAt = createdAt; }
}

class ReviewStatisticsVO {
    private Long totalReviews;
    private Double averageRating;
    private Integer fiveStarCount;
    private Integer fourStarCount;
    private Integer threeStarCount;
    private Integer twoStarCount;
    private Integer oneStarCount;

    public Long getTotalReviews() { return totalReviews; }
    public void setTotalReviews(Long totalReviews) { this.totalReviews = totalReviews; }
    public Double getAverageRating() { return averageRating; }
    public void setAverageRating(Double averageRating) { this.averageRating = averageRating; }
    public Integer getFiveStarCount() { return fiveStarCount; }
    public void setFiveStarCount(Integer fiveStarCount) { this.fiveStarCount = fiveStarCount; }
    public Integer getFourStarCount() { return fourStarCount; }
    public void setFourStarCount(Integer fourStarCount) { this.fourStarCount = fourStarCount; }
    public Integer getThreeStarCount() { return threeStarCount; }
    public void setThreeStarCount(Integer threeStarCount) { this.threeStarCount = threeStarCount; }
    public Integer getTwoStarCount() { return twoStarCount; }
    public void setTwoStarCount(Integer twoStarCount) { this.twoStarCount = twoStarCount; }
    public Integer getOneStarCount() { return oneStarCount; }
    public void setOneStarCount(Integer oneStarCount) { this.oneStarCount = oneStarCount; }
}

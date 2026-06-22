package com.example.controller;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import com.example.model.Review;
import com.example.model.Result;
import com.example.model.PageResult;
import java.util.List;

@Path("/api/reviews")
public class ReviewController extends BaseCrudController<CreateReviewReq, Review> {

    @GET
    @Path("/product/{productId}")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<PageResult<Review>> getReviewsByProduct(
            @PathParam("productId") Long productId,
            @QueryParam("rating") Integer rating,
            @DefaultValue("1") @QueryParam("pageNum") Integer pageNum,
            @DefaultValue("10") @QueryParam("pageSize") Integer pageSize) {
        return Result.success(new PageResult<>());
    }

    @POST
    @Path("/{id}/comments")
    @Consumes(MediaType.APPLICATION_JSON)
    @Produces(MediaType.APPLICATION_JSON)
    public Result<CommentVO> addComment(@PathParam("id") Long id, CreateCommentReq req) {
        return Result.success(new CommentVO());
    }

    @GET
    @Path("/{id}/comments")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<CommentVO>> getComments(@PathParam("id") Long id) {
        return Result.success(List.of());
    }

    @PUT
    @Path("/{id}/moderate")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<Review> moderateReview(@PathParam("id") Long id, @QueryParam("status") Integer status) {
        return Result.success(new Review());
    }

    @GET
    @Path("/user/{userId}")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<PageResult<Review>> getReviewsByUser(
            @PathParam("userId") Long userId,
            @DefaultValue("1") @QueryParam("pageNum") Integer pageNum,
            @DefaultValue("10") @QueryParam("pageSize") Integer pageSize) {
        return Result.success(new PageResult<>());
    }

    @GET
    @Path("/statistics")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<ReviewStatisticsVO> getStatistics(@QueryParam("productId") Long productId) {
        return Result.success(new ReviewStatisticsVO());
    }
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

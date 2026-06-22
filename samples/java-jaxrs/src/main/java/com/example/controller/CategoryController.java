package com.example.controller;

import javax.ws.rs.*;
import javax.ws.rs.core.MediaType;
import com.example.model.Category;
import com.example.model.Result;
import com.example.model.PageResult;
import java.util.List;

@Path("/api/categories")
public class CategoryController extends BaseCrudController<CreateCategoryReq, Category> {

    @GET
    @Path("/tree")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<CategoryTreeVO>> getCategoryTree() {
        return Result.success(List.of());
    }

    @GET
    @Path("/{id}/products")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<PageResult<Object>> getProductsByCategory(
            @PathParam("id") Long id,
            @DefaultValue("1") @QueryParam("pageNum") Integer pageNum,
            @DefaultValue("10") @QueryParam("pageSize") Integer pageSize) {
        return Result.success(new PageResult<>());
    }

    @PUT
    @Path("/{id}/sort")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<Category> updateSort(@PathParam("id") Long id, @QueryParam("sort") Integer sort) {
        return Result.success(new Category());
    }

    @GET
    @Path("/roots")
    @Produces(MediaType.APPLICATION_JSON)
    public Result<List<Category>> getRootCategories() {
        return Result.success(List.of());
    }
}

class CreateCategoryReq {
    private String name;
    private Long parentId;
    private Integer sort;
    private String icon;

    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public Long getParentId() { return parentId; }
    public void setParentId(Long parentId) { this.parentId = parentId; }
    public Integer getSort() { return sort; }
    public void setSort(Integer sort) { this.sort = sort; }
    public String getIcon() { return icon; }
    public void setIcon(String icon) { this.icon = icon; }
}

class CategoryTreeVO {
    private Long id;
    private String name;
    private Long parentId;
    private Integer sort;
    private String icon;
    private List<CategoryTreeVO> children;

    public Long getId() { return id; }
    public void setId(Long id) { this.id = id; }
    public String getName() { return name; }
    public void setName(String name) { this.name = name; }
    public Long getParentId() { return parentId; }
    public void setParentId(Long parentId) { this.parentId = parentId; }
    public Integer getSort() { return sort; }
    public void setSort(Integer sort) { this.sort = sort; }
    public String getIcon() { return icon; }
    public void setIcon(String icon) { this.icon = icon; }
    public List<CategoryTreeVO> getChildren() { return children; }
    public void setChildren(List<CategoryTreeVO> children) { this.children = children; }
}

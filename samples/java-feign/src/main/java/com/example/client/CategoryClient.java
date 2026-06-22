package com.example.client;

import feign.Param;
import feign.RequestLine;
import com.example.model.Category;
import com.example.model.Result;
import com.example.model.PageResult;
import java.util.List;

public interface CategoryClient extends BaseCrudClient<CreateCategoryReq, Category> {

    @RequestLine("GET /api/categories/tree")
    Result<List<CategoryTreeVO>> getCategoryTree();

    @RequestLine("GET /api/categories/{id}/products?pageNum={pageNum}&pageSize={pageSize}")
    Result<PageResult<Object>> getProductsByCategory(@Param("id") Long id,
                                                      @Param("pageNum") Integer pageNum,
                                                      @Param("pageSize") Integer pageSize);

    @RequestLine("PUT /api/categories/{id}/sort?sort={sort}")
    Result<Category> updateSort(@Param("id") Long id, @Param("sort") Integer sort);

    @RequestLine("GET /api/categories/roots")
    Result<List<Category>> getRootCategories();
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

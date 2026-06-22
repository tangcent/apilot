package com.example.controller;

import org.springframework.web.bind.annotation.*;
import com.example.model.Category;
import com.example.model.Result;
import com.example.model.PageResult;
import java.util.List;

@RestController
@RequestMapping("/api/categories")
public class CategoryController extends BaseCrudController<CreateCategoryReq, Category> {

    @GetMapping("/tree")
    public Result<List<CategoryTreeVO>> getCategoryTree() {
        return Result.success(List.of());
    }

    @GetMapping("/{id}/products")
    public Result<PageResult<Object>> getProductsByCategory(
            @PathVariable Long id,
            @RequestParam(defaultValue = "1") Integer pageNum,
            @RequestParam(defaultValue = "10") Integer pageSize) {
        return Result.success(new PageResult<>());
    }

    @PutMapping("/{id}/sort")
    public Result<Category> updateSort(@PathVariable Long id, @RequestParam Integer sort) {
        return Result.success(new Category());
    }

    @GetMapping("/roots")
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

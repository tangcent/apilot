package com.example.model;

import java.util.List;

public class PageResult<T> {
    private Long total;
    private Integer page;
    private Integer size;
    private List<T> items;

    public Long getTotal() { return total; }
    public void setTotal(Long total) { this.total = total; }
    public Integer getPage() { return page; }
    public void setPage(Integer page) { this.page = page; }
    public Integer getSize() { return size; }
    public void setSize(Integer size) { this.size = size; }
    public List<T> getItems() { return items; }
    public void setItems(List<T> items) { this.items = items; }
}

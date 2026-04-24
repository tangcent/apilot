package com.example.demo.model;

public class TypedEntity<T> extends BaseEntity {
    private T typeInfo;

    public T getTypeInfo() { return typeInfo; }
    public void setTypeInfo(T typeInfo) { this.typeInfo = typeInfo; }
}

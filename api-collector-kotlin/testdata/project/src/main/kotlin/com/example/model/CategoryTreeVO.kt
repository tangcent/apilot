package com.example.model

data class CategoryTreeVO(
    val id: Long,
    val name: String,
    val parentId: Long?,
    val sort: Int?,
    val icon: String?,
    val children: List<CategoryTreeVO>? = null,
)

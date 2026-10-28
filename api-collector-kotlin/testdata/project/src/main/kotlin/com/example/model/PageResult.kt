package com.example.model

data class PageResult<T>(
    val total: Long,
    val page: Int,
    val size: Int,
    val items: List<T>,
)

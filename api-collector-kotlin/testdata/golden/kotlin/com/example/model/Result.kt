package com.example.model

data class Result<T>(
    val code: Int,
    val message: String?,
    val data: T?,
)

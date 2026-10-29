package com.example.model

data class Result<T>(
    val code: Int,
    val message: String? = null,
    val data: T? = null,
)

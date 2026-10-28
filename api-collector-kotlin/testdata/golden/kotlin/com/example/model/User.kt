package com.example.model

data class User(
    val username: String,
    val email: String?,
    val password: String?,
    val phone: String?,
    val avatar: String?,
    val status: Int?,
) : BaseEntity()

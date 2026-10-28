package com.example.model

data class CreateUserReq(
    val username: String,
    val password: String,
    val email: String? = null,
)

data class UpdateUserReq(
    val email: String? = null,
    val phone: String? = null,
)

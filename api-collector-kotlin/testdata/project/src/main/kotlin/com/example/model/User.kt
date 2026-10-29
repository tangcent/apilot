package com.example.model

import io.swagger.v3.oas.annotations.media.Schema

data class User(
    @Schema(description = "login name", example = "alice") val username: String,
    val email: String?,
    val password: String?,
    val phone: String?,
    val avatar: String?,
    val status: Int?,
) : BaseEntity()

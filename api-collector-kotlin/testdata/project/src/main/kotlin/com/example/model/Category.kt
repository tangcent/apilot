package com.example.model

data class Category(
    val id: Long?,
    val name: String,
    val parentId: Long? = null,
    val sort: Int? = null,
    val icon: String? = null,
    val status: Status = Status.ACTIVE,
) : BaseEntity()

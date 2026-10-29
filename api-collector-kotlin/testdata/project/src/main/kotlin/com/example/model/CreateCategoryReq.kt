package com.example.model

import com.fasterxml.jackson.annotation.JsonIgnore
import com.fasterxml.jackson.annotation.JsonProperty
import io.swagger.v3.oas.annotations.media.Schema
import org.jetbrains.annotations.Nullable

data class CreateCategoryReq(
    val name: String,
    @Nullable val parentId: Long? = null,
    @JsonProperty("display_name") val displayName: String? = null,
    @JsonIgnore val secret: String? = null,
    @Schema(description = "sort order", example = "1") val sort: Int? = null,
)

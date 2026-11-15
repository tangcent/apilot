package com.example.recovery

import jakarta.validation.Constraint
import jakarta.validation.Payload
import kotlin.reflect.KClass

@Constraint(validatedBy = [TypeSizeValidator::class])
annotation class TypeSize(
    val message: String = "size",
    val groups: Array<KClass<*>> = [],
    val payload: Array<KClass<out Payload>> = []
)

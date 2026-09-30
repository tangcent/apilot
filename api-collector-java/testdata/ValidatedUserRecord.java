package com.example.demo.model;

import jakarta.validation.constraints.Max;
import jakarta.validation.constraints.NotNull;
import jakarta.validation.constraints.Size;

/**
 * A user payload validated through record component annotations.
 */
public record ValidatedUserRecord(
        @NotNull @Size(min = 2, max = 50) String name,
        @NotNull String email,
        @Max(150) int age) {

    public ValidatedUserRecord {
        if (age < 0) {
            throw new IllegalArgumentException("age must be non-negative");
        }
    }

    public String displayName() {
        return name + " <" + email + ">";
    }
}

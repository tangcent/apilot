package com.example.arrayfield;

public class ArrayFieldBox<T> {
    /** The array of T is the argument of the wrapper, which binds it to U. */
    private ArrayFieldWrapper<T[]> payload;
}

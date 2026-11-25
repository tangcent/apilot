package com.example.jdkidentity.lib;

import java.util.List;

public class Outer<T> {
    private Wrapper<T> wrapped;
    private Wrapper<List<T>> wrappedList;
    private Wrapper<T[]> wrappedArray;
}

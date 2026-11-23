package com.example.jdkidentity.lib;

import java.util.List;
import java.util.Map;

public class Box<T> {
    private T data;
    private List<T> list;
    private T[] array;
    private Map<java.lang.String, T> byKey;
    private List<? extends T> bounded;
}

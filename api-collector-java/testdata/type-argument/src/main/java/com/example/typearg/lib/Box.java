package com.example.typearg.lib;

import java.util.List;
import java.util.Map;

public class Box<T> {
    private T data;
    private List<T> list;
    private T[] array;
    private Map<String, T> byKey;
    private List<? extends T> bounded;
}

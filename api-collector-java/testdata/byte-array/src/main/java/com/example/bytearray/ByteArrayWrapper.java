package com.example.bytearray;

import java.util.List;
import java.util.Map;

/** Takes its type argument as a field, an element, an array and a map value. */
public class ByteArrayWrapper<U> {
    private U data;
    private List<U> items;
    private U[] grid;
    private Map<String, U> byName;
}

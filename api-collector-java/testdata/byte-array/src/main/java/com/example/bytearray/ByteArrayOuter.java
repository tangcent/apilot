package com.example.bytearray;

import java.util.List;

/** Hands its type argument on to a generic class, as a field and as an element. */
public class ByteArrayOuter<T> {
    private ByteArrayWrapper<T> wrapped;
    private List<ByteArrayWrapper<T>> wrappedItems;
    private T direct;
}

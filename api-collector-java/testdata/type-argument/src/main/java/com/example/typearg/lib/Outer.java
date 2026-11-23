package com.example.typearg.lib;

import java.util.List;

public class Outer<T> {
    /** The parameter is handed on to the next generic class. */
    private Wrapper<T> wrapped;
    private Wrapper<List<T>> wrappedList;
    private Wrapper<T[]> wrappedArray;
}

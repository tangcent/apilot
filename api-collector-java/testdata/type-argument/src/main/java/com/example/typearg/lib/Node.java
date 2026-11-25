package com.example.typearg.lib;

/** A type that refers to itself. */
public class Node<T> {
    private T value;
    private Node<T> next;
}

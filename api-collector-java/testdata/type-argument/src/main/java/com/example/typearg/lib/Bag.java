package com.example.typearg.lib;

import java.util.HashMap;
import java.util.List;

/** A map whose value is a list of its parameter. */
public class Bag<V> extends HashMap<String, List<V>> {
}

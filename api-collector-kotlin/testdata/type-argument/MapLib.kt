package com.example.typearg.lib

import com.example.typearg.dto.Item

/** A map whose value is a list of its parameter. */
class Bag<V> : HashMap<String, List<V>>()

/** Written in the library's package, which has an Item of its own: the imported one wins. */
class Named : HashMap<String, Item>()

/** The value is the second parameter. */
class Pairs<K, V> : HashMap<K, V>()

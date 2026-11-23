package com.example.typearg.lib;

import java.util.List;

public class Holder {
    /** Written in full: the application's class, though the library has an Item too. */
    private List<com.example.typearg.dto.Item> fixedList;
    private Box<com.example.typearg.dto.Item> fixedBox;

    /** Written simply: the library's own class. */
    private Item own;
}

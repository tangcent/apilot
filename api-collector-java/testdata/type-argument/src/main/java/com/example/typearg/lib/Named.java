package com.example.typearg.lib;

import com.example.typearg.dto.Item;
import java.util.HashMap;

/** Written in the library's package, which has an Item of its own: the imported one wins. */
public class Named extends HashMap<String, Item> {
}

package com.example.typearg.app;

import com.example.typearg.dto.Item;
import com.example.typearg.lib.BaseController;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/items")
public class ItemController extends BaseController<Item> {
}

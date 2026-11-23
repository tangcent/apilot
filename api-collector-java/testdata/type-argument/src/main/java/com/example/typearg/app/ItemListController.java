package com.example.typearg.app;

import com.example.typearg.dto.Item;
import com.example.typearg.lib.MidController;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/lists")
public class ItemListController extends MidController<Item> {
}

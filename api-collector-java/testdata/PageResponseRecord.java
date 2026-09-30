package com.example.demo.model;

import java.util.List;

public record PageResponseRecord<T>(List<T> items, int page, int total) {}

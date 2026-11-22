package com.example.arrayfield;

import java.util.List;
import javax.validation.constraints.NotEmpty;

public class ArrayFieldForm {
    private String[] names;
    private int[] ids;
    private Long[] longs;
    private ArrayFieldItem[] items;
    private List<String>[] groups;
    private int[][] grid;

    /** The brackets may follow the name. */
    private String legacy[];

    /** Spaces may stand in front of the brackets and between them. */
    private List<String []> spaced;
    private List<int [ ]> spacedIds;

    /** A name qualified with its package is the same type as the simple one. */
    private java.time.LocalDate day;
    private java.math.BigDecimal[] amounts;

    @NotEmpty
    private String[] codes;
}

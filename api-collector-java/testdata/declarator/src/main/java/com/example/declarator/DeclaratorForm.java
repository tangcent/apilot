package com.example.declarator;

import java.util.List;
import javax.validation.constraints.NotBlank;

public class DeclaratorForm {
    /** The name of a person. */
    @NotBlank
    private String firstName, lastName;

    private int width = 1, height, depth[];

    private String note = "n", tags[] = {"t"}, remark;

    private List<String> aliases, nicknames;

    private static final String KIND = "kind", LEVEL = "level";
}

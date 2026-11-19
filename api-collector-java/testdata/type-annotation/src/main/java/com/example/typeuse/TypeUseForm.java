package com.example.typeuse;

import java.util.List;
import java.util.Map;
import javax.validation.constraints.NotBlank;
import javax.validation.constraints.NotNull;

public class TypeUseForm {
    private List<@NotBlank(message = Msgs.CANNOT_BE_EMPTY) String> names;

    private Map<String, List<@Patterns(regexps = {Res.HTTPS_URL, "a, b"}) String>> urls;

    private List<@NotNull(message = "id") Long> ids;

    /** A constraint on the field itself still makes it required. */
    @NotNull
    private List<@NotBlank String> tags;

    private TypeUseItem item;
}

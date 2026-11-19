package com.example.typeuse;

import java.util.List;
import javax.validation.constraints.NotBlank;

public class TypeUseItem {
    private List<@NotBlank(message = Msgs.CANNOT_BE_EMPTY) String> labels;
}

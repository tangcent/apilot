package com.example.forms;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.annotation.JsonProperty;
import com.fasterxml.jackson.annotation.JsonValue;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;

/** Jackson writes the name of the constant. */
enum Plain {
    A,
    B
}

/** Jackson writes the name @JsonProperty gives. */
enum Renamed {
    A,
    @JsonProperty("b_x")
    B
}

interface Coded {
    @JsonValue
    String code();
}

/** Jackson writes what the method of the interface returns. */
enum ViaCoded implements Coded {
    A;

    public String code() {
        return "a";
    }
}

/** Jackson writes the index of the constant. */
@JsonFormat(shape = JsonFormat.Shape.NUMBER_INT)
enum Indexed {
    /** First. */
    X,
    Y
}

/** Jackson writes the index of the constant. */
@JsonFormat(shape = JsonFormat.Shape.ARRAY)
enum Arrayed {
    A
}

/** Jackson writes what the serializer writes. */
@JsonSerialize(using = FormWriter.class)
enum Serialized {
    A
}

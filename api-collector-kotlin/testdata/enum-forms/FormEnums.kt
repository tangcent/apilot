package com.example.forms

import com.fasterxml.jackson.annotation.JsonFormat
import com.fasterxml.jackson.annotation.JsonProperty
import com.fasterxml.jackson.annotation.JsonValue
import com.fasterxml.jackson.databind.annotation.JsonSerialize

/** Jackson writes the name of the entry. */
enum class Plain {
    A,
    B
}

/** Jackson writes the name @JsonProperty gives. */
enum class Renamed {
    A,
    @JsonProperty("b_x") B
}

interface Coded {
    @get:JsonValue
    val wire: String
}

/** Jackson writes what the member of the interface returns. */
enum class ViaCoded : Coded {
    A;

    override val wire: String get() = "a"
}

/** Jackson writes the index of the entry. */
@JsonFormat(shape = JsonFormat.Shape.NUMBER_INT)
enum class Indexed {
    /** First. */
    X,
    Y
}

/** Jackson writes the index of the entry. */
@JsonFormat(shape = JsonFormat.Shape.ARRAY)
enum class Arrayed {
    A,
    B
}

/** Jackson writes what the serializer writes. */
@JsonSerialize(using = FormWriter::class)
enum class Serialized {
    A,
    B
}

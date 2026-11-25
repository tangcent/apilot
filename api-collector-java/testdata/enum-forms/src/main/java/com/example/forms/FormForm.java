package com.example.forms;

import com.fasterxml.jackson.annotation.JsonFormat;
import com.fasterxml.jackson.databind.annotation.JsonSerialize;
import java.util.List;

public class FormForm {
    private Plain plain;

    private Renamed renamed;

    private List<Renamed> renamedList;

    private ViaCoded viaCoded;

    private Indexed indexed;

    private Arrayed arrayed;

    private Serialized serialized;

    @JsonFormat(shape = JsonFormat.Shape.NUMBER)
    private Plain fieldIndex;

    @JsonFormat(shape = JsonFormat.Shape.NUMBER_INT)
    private List<Plain> listIndex;

    @JsonSerialize(using = FormWriter.class)
    private Plain fieldSerialized;
}

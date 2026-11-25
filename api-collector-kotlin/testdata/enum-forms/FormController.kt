package com.example.forms

import com.fasterxml.jackson.annotation.JsonFormat
import com.fasterxml.jackson.databind.annotation.JsonSerialize
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

class FormForm {
    var plain: Plain? = null

    var renamed: Renamed? = null

    var renamedList: List<Renamed>? = null

    var viaCoded: ViaCoded? = null

    var indexed: Indexed? = null

    var arrayed: Arrayed? = null

    var serialized: Serialized? = null

    @JsonFormat(shape = JsonFormat.Shape.NUMBER)
    var fieldIndex: Plain? = null

    @JsonFormat(shape = JsonFormat.Shape.NUMBER_INT)
    var listIndex: List<Plain>? = null

    @JsonSerialize(using = FormWriter::class)
    var fieldSerialized: Plain? = null
}

@RestController
@RequestMapping("/forms")
class FormController {

    @PostMapping
    fun create(@RequestBody form: FormForm) {
    }

    @GetMapping
    fun page(query: FormForm) {
    }
}

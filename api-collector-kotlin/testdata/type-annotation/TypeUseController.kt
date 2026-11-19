package com.example.typeuse

import javax.validation.constraints.NotBlank
import javax.validation.constraints.NotNull
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

class TypeUseItem(
    val labels: List<@NotBlank(message = Msgs.CANNOT_BE_EMPTY) String>?,
)

class TypeUseForm(
    val names: List<@NotBlank(message = Msgs.CANNOT_BE_EMPTY) String>?,
    val urls: Map<String, List<@Patterns(regexps = [Res.HTTPS_URL, "a, b"]) String>>?,
    val ids: List<@NotNull(message = "id") Long>?,
    /** A constraint on the property itself still makes it required. */
    @field:NotNull
    val tags: List<@NotBlank String>?,
    val item: TypeUseItem?,
)

@RestController
@RequestMapping("/type-use")
class TypeUseController {

    @PostMapping("/save")
    fun save(@RequestBody form: TypeUseForm): TypeUseForm = form
}

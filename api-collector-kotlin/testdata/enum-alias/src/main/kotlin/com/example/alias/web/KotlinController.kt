package com.example.alias.web

import com.example.alias.decoy.DecoyJson
import com.example.alias.decoy.DecoyPlain
import com.example.alias.use.*
import org.springframework.web.bind.annotation.GetMapping
import org.springframework.web.bind.annotation.PostMapping
import org.springframework.web.bind.annotation.RequestBody
import org.springframework.web.bind.annotation.RequestMapping
import org.springframework.web.bind.annotation.RestController

/** Each property is typed with an enum that implements an interface, one with @JsonValue and one without. */
class KotlinForm {
    var renamed: RenamedJson? = null
    var renamedPlain: RenamedPlain? = null

    var generic: GenericJson? = null
    var genericPlain: GenericPlain? = null

    var nestedRenamed: NestedRenamedJson? = null
    var nestedRenamedPlain: NestedRenamedPlain? = null

    var chain: ChainJson? = null
    var chainPlain: ChainPlain? = null

    var javaInterface: JavaJson? = null
    var javaInterfacePlain: JavaPlain? = null

    var imported: ImportedJson? = null
    var importedPlain: ImportedPlain? = null

    var samePackage: SameJson? = null
    var samePackagePlain: SamePlain? = null

    var onDemand: StarJson? = null
    var onDemandPlain: StarPlain? = null

    var qualified: QualifiedJson? = null
    var qualifiedPlain: QualifiedPlain? = null

    var outer: OuterJson? = null
    var outerPlain: OuterPlain? = null

    var typealias: TypealiasJson? = null
    var typealiasPlain: TypealiasPlain? = null

    var decoy: DecoyJson? = null
    var decoyPlain: DecoyPlain? = null

    var boxed: Box.Kind? = null
    var boxedPlain: Box.Calm? = null
}

@RestController
@RequestMapping("/kotlin")
class KotlinController {

    @PostMapping
    fun create(@RequestBody form: KotlinForm) {
    }

    @GetMapping
    fun page(query: KotlinForm) {
    }
}

package com.example.alias.web;

import com.example.alias.decoy.DecoyJson;
import com.example.alias.decoy.DecoyPlain;
import com.example.alias.use.*;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

/** A holder written in Java, typed with the enums written in Kotlin. */
class JavaForm {
    private RenamedJson renamed;
    private RenamedPlain renamedPlain;

    private GenericJson generic;
    private GenericPlain genericPlain;

    private NestedRenamedJson nestedRenamed;
    private NestedRenamedPlain nestedRenamedPlain;

    private ChainJson chain;
    private ChainPlain chainPlain;

    private JavaJson javaInterface;
    private JavaPlain javaInterfacePlain;

    private ImportedJson imported;
    private ImportedPlain importedPlain;

    private SameJson samePackage;
    private SamePlain samePackagePlain;

    private StarJson onDemand;
    private StarPlain onDemandPlain;

    private QualifiedJson qualified;
    private QualifiedPlain qualifiedPlain;

    private OuterJson outer;
    private OuterPlain outerPlain;

    private TypealiasJson typealias;
    private TypealiasPlain typealiasPlain;

    private DecoyJson decoy;
    private DecoyPlain decoyPlain;

    private Box.Kind boxed;
    private Box.Calm boxedPlain;
}

@RestController
@RequestMapping("/java")
public class JavaController {

    @PostMapping
    public void create(@RequestBody JavaForm form) {
    }

    @GetMapping
    public void page(JavaForm query) {
    }
}

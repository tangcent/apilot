package com.example.langscope;

import com.example.langscope.imported.ImportedForm;
import com.example.langscope.local.LocalForm;
import com.example.langscope.plain.PlainForm;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping("/lang-scope")
public class LangScopeController {

    @PostMapping("/imported")
    public ImportedForm imported(@RequestBody ImportedForm form) {
        return form;
    }

    @PostMapping("/local")
    public LocalForm local(@RequestBody LocalForm form) {
        return form;
    }

    @PostMapping("/plain")
    public PlainForm plain(@RequestBody PlainForm form) {
        return form;
    }
}

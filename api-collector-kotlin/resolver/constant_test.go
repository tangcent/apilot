package resolver

import (
	"testing"

	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	model "github.com/tangcent/apilot/api-model"
)

func constVal(name, typ, initializer string) ktparser.Property {
	return ktparser.Property{Name: name, Type: typ, IsConst: true, HasDefault: true, DefaultValue: initializer}
}

func TestConstValsAreStaticFieldsWithTheirInitializer(t *testing.T) {
	class := ktparser.Class{
		Name:    "Response",
		Package: "com.example.model",
		Fields: []ktparser.Property{
			testProperty("code", "Int"),
			constVal("OK", "", `"ok"`),
			constVal("TYPED", "String", `"typed"`),
			constVal("ALIAS", "", "Base.OK"),
			constVal("ESCAPED", "", `"\$x"`),
			constVal("RAW", "", `"""raw"""`),
			constVal("TEMPLATE", "", `"$OK-x"`),
			constVal("BRACED", "", `"${OK}"`),
			constVal("LIMIT", "Int", "10"),
		},
	}
	want := map[string]string{
		"OK":       `"ok"`,
		"TYPED":    `"typed"`,
		"ALIAS":    "Base.OK",
		"ESCAPED":  `"\$x"`,
		"RAW":      `"raw"`,
		"TEMPLATE": "",
		"BRACED":   "",
		"LIMIT":    "",
	}
	for _, field := range newTypeScope([]ktparser.Class{class}, nil).toJavaClass(class).Fields {
		if field.Name == "code" {
			if field.IsStatic || field.Constant != "" {
				t.Errorf("code = %+v, want an instance field", field)
			}
			continue
		}
		if !field.IsStatic || !field.IsFinal {
			t.Errorf("%s: static %v, final %v; a const val is a static final field", field.Name, field.IsStatic, field.IsFinal)
		}
		if field.Constant != want[field.Name] {
			t.Errorf("%s: constant = %q, want %q", field.Name, field.Constant, want[field.Name])
		}
	}

	// A const val is no JSON property.
	resolved := New([]ktparser.Class{class}).ResolveFrom(&class, "Response", nil)
	if resolved.Kind != model.KindObject || len(resolved.Fields) != 1 || resolved.Fields["code"] == nil {
		t.Errorf("Response = %v, want only code", fieldNames(resolved))
	}
}

func TestConstantValueFindsTopLevelConstants(t *testing.T) {
	facade := ktparser.Class{
		Name: "TenantsKt", Package: "com.example.shared", FilePath: "shared/Tenants.kt", IsFileFacade: true,
		Fields: []ktparser.Property{constVal("TENANT", "", `"tenant_id"`), constVal("ALIAS", "", "TENANT")},
	}
	local := ktparser.Class{
		Name: "RegionsKt", Package: "com.example.app", FilePath: "Regions.kt", IsFileFacade: true,
		Fields: []ktparser.Property{constVal("REGION", "", `"region"`)},
	}
	starred := ktparser.Class{
		Name: "ZonesKt", Package: "com.example.zones", FilePath: "zones/Zones.kt", IsFileFacade: true,
		Fields: []ktparser.Property{constVal("ZONE", "", `"zone"`)},
	}
	source := ktparser.Class{
		Name: "Handler", Package: "com.example.app", FilePath: "Handler.kt",
		Imports: []ktparser.Import{{Path: "com.example.shared.TENANT"}, {Path: "com.example.zones", OnDemand: true}},
	}
	r := New([]ktparser.Class{facade, local, starred, source})

	for ref, want := range map[string]string{
		"TENANT":                    "tenant_id",
		"REGION":                    "region",
		"ZONE":                      "zone",
		"com.example.shared.ALIAS":  "tenant_id",
		"com.example.shared.TENANT": "tenant_id",
	} {
		if got, ok := r.ConstantValue(&source, ref); !ok || got != want {
			t.Errorf("ConstantValue(%q) = %q, %v; want %q", ref, got, ok, want)
		}
	}
	if got, ok := r.ConstantValue(&source, "ALIAS"); ok {
		t.Errorf("ConstantValue(ALIAS) = %q: a top-level constant of another package is not in scope unimported", got)
	}
}

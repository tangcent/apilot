package kotlincollector

import (
	"path/filepath"
	"strings"
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	javaparser "github.com/tangcent/apilot/api-collector-java/parser"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
	"github.com/tangcent/apilot/api-collector-kotlin/springmvc"
	model "github.com/tangcent/apilot/api-model"
)

// Kotlin sources of a shared library, as the indexer returns them from its
// sources JAR.
var librarySources = map[string]string{
	"com.example.lib.Wrapper": `package com.example.lib

/** Response envelope. */
class Wrapper<T> {
    /** Business code. */
    var code: Int = 0
    var data: T? = null
}
`,
	"com.example.lib.PageData": `package com.example.lib

class PageData<T>(
    val total: Long,
    val rows: List<T>,
)
`,
	"com.example.lib.PageQuery": `package com.example.lib

import javax.validation.constraints.NotNull

class PageQuery {
    /** Page number. */
    var page: Int = 1

    @field:NotNull
    var size: Int? = null
}
`,
	"com.example.lib.OrderState": `package com.example.lib

/**
 * State of an order.
 */
enum class OrderState(val code: String) {
    /** Waiting for payment. */
    PENDING("P"),
    /**
     * Paid.
     *
     * Ships within a day.
     */
    PAID("D"),
    CLOSED("C"),
    ;

    companion object {
        /** The state a code stands for. */
        @JvmStatic
        fun of(code: String?): OrderState? = values().firstOrNull { it.code == code }
    }
}
`,
	// Currency declares @JsonValue, so a property typed with it takes what the
	// getter returns, not the name of an entry (see enum_test.go).
	"com.example.lib.Currency": `package com.example.lib

import com.fasterxml.jackson.annotation.JsonValue

enum class Currency(@get:JsonValue val symbol: String) {
    USD("usd"),
    EUR("eur"),
}
`,
}

// decompiledLibrarySources are classes of the same library that the indexer
// can only decompile, for want of a sources JAR: Java without doc comments.
// The decompiler prints a Java enum as an enum, and a Kotlin enum as a class
// that extends Enum.
var decompiledLibrarySources = map[string]string{
	"com.example.lib.Region": `/*
 * Decompiled with CFR 0.152.
 */
package com.example.lib;

import kotlin.Metadata;
import kotlin.enums.EnumEntries;
import kotlin.enums.EnumEntriesKt;
import org.jetbrains.annotations.NotNull;

@Metadata(mv={1, 9, 0}, k=1, xi=48)
public final class Region
extends Enum<Region> {
    public static final /* enum */ Region NORTH = new Region("n");
    public static final /* enum */ Region SOUTH = new Region("s");
    private static final /* synthetic */ Region[] $VALUES;
    private static final /* synthetic */ EnumEntries $ENTRIES;
    @NotNull
    private final String code;

    private Region(String code) {
        this.code = code;
    }

    @NotNull
    public final String getCode() {
        return this.code;
    }

    public static Region[] values() {
        return (Region[])$VALUES.clone();
    }

    public static Region valueOf(String value) {
        return Enum.valueOf(Region.class, value);
    }

    @NotNull
    public static EnumEntries<Region> getEntries() {
        return $ENTRIES;
    }

    static {
        $VALUES = regionArray = new Region[]{Region.NORTH, Region.SOUTH};
        $ENTRIES = EnumEntriesKt.enumEntries((Enum[])$VALUES);
    }
}
`,
	"com.example.lib.PayChannel": `/*
 * Decompiled with CFR 0.152.
 */
package com.example.lib;

public enum PayChannel {
    ALIPAY("alipay"),
    WECHAT("wechat");

    private final String code;

    private PayChannel(String code) {
        this.code = code;
    }

    public String getCode() {
        return this.code;
    }
}
`,
	// Tier is a Kotlin enum with @JsonValue on its getter, decompiled like
	// Region.
	"com.example.lib.Tier": `/*
 * Decompiled with CFR 0.152.
 */
package com.example.lib;

import com.fasterxml.jackson.annotation.JsonValue;
import kotlin.Metadata;
import org.jetbrains.annotations.NotNull;

@Metadata(mv={1, 9, 0}, k=1, xi=48)
public final class Tier
extends Enum<Tier> {
    public static final /* enum */ Tier GOLD = new Tier("gold");
    public static final /* enum */ Tier SILVER = new Tier("silver");
    private static final /* synthetic */ Tier[] $VALUES;
    @NotNull
    private final String code;

    private Tier(String code) {
        this.code = code;
    }

    @JsonValue
    @NotNull
    public final String getCode() {
        return this.code;
    }

    static {
        $VALUES = tierArray = new Tier[]{Tier.GOLD, Tier.SILVER};
    }
}
`,
}

// libraryResolver answers dependency lookups from librarySources through the
// parser the collector registers for Kotlin dependency sources, and from
// decompiledLibrarySources through the Java parser.
type libraryResolver struct {
	t       *testing.T
	queried []string
}

func (r *libraryResolver) ResolveClass(className string) *javaparser.Class {
	r.queried = append(r.queried, className)
	var classes []javaparser.Class
	var err error
	if source, ok := decompiledLibrarySources[className]; ok {
		classes, err = parseJavaSource(source)
	} else if source, ok := librarySources[className]; ok {
		classes, err = parseDependencySource(className, []byte(source))
	} else {
		return nil
	}
	if err != nil {
		r.t.Fatalf("parsing %s: %v", className, err)
	}
	simple := className[strings.LastIndex(className, ".")+1:]
	for i := range classes {
		if classes[i].Name == simple {
			return &classes[i]
		}
	}
	return nil
}

func parseJavaSource(source string) ([]javaparser.Class, error) {
	p, err := javaparser.NewParser(javaparser.ParserOptions{})
	if err != nil {
		return nil, err
	}
	defer p.Close()
	return p.ParseSource([]byte(source))
}

func TestParseDependencySourceReadsKotlinLikeTheProject(t *testing.T) {
	classes, err := parseDependencySource("com.example.lib.Wrapper", []byte(librarySources["com.example.lib.Wrapper"]))
	if err != nil {
		t.Fatal(err)
	}
	if len(classes) != 1 || classes[0].Name != "Wrapper" || classes[0].Package != "com.example.lib" {
		t.Fatalf("classes = %+v", classes)
	}
	wrapper := classes[0]
	if len(wrapper.TypeParameters) != 1 || wrapper.TypeParameters[0] != "T" {
		t.Errorf("type parameters = %v", wrapper.TypeParameters)
	}
	if len(wrapper.Fields) != 2 || wrapper.Fields[0].JavaDoc != "Business code." {
		t.Errorf("fields = %+v", wrapper.Fields)
	}

	// A dependency source gets the project's grammar-error recovery: the
	// damaged declaration is skipped and the rest of the source still reads.
	damaged, err := parseDependencySource("Broken", []byte("package com.example.lib\n\nclass Broken(val a: Int = )\n\nclass Fine(val b: String)\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(damaged) != 1 || damaged[0].Name != "Fine" {
		t.Errorf("classes of a damaged source = %+v, want only Fine", damaged)
	}
}

func TestKotlinEndpointResolvesTypesOfAKotlinLibrary(t *testing.T) {
	p, err := ktparser.NewParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	dir, _ := filepath.Abs("testdata/dependency")
	results, err := p.ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	typeResolver := resolver.New(flattenClasses(results))
	library := &libraryResolver{t: t}
	typeResolver.SetDependencyResolver(library)

	var response *model.ObjectModel
	for _, ctrl := range springmvc.NewParser(typeResolver).ExtractControllers(results) {
		for _, ep := range ctrl.Endpoints {
			if ep.Path == "/orders" {
				response = ep.ResponseSchema
			}
		}
	}

	objectField(t, response, "code")
	data := objectField(t, response, "data")
	objectField(t, data.Model, "total")
	rows := objectField(t, data.Model, "rows")
	if rows.Model.Kind != model.KindArray {
		t.Fatalf("rows = %+v", rows.Model)
	}
	objectField(t, rows.Model.Items, "id")
	objectField(t, rows.Model.Items, "no")
	if status := objectField(t, rows.Model.Items, "status"); status.Model.TypeName != model.JsonTypeString || len(status.Options) != 2 {
		t.Errorf("status = %+v, options %v: a project enum is a string of its entries", status.Model, status.Options)
	}

	if unresolved := typeResolver.Unresolved(); len(unresolved) != 0 {
		t.Errorf("unresolved = %v", unresolved)
	}
	for _, want := range []string{"com.example.lib.Wrapper", "com.example.lib.PageData"} {
		found := false
		for _, q := range library.queried {
			found = found || q == want
		}
		if !found {
			t.Errorf("the library was not asked for %s; asked for %v", want, library.queried)
		}
	}
	// The project's own enum is not a dependency type: asking the index would
	// read an installed release of the project.
	for _, q := range library.queried {
		if q == "com.example.app.OrderStatus" {
			t.Errorf("the library was asked for the project's enum; asked for %v", library.queried)
		}
	}
}

// An unannotated parameter is bound from request parameters through the
// binding view, which is built separately from the JSON view. It must expand
// a dependency type too, and follow the --required-fields mode.
func TestKotlinBindingViewResolvesTypesOfALibrary(t *testing.T) {
	p, err := ktparser.NewParser()
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	dir, _ := filepath.Abs("testdata/dependency")
	results, err := p.ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		mode string
		want map[string]bool
	}{
		{collector.RequiredFieldsDefault, map[string]bool{"page": true, "size": true}},
		{collector.RequiredFieldsConstraints, map[string]bool{"page": false, "size": true}},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			typeResolver := resolver.New(flattenClasses(results))
			typeResolver.SetDependencyResolver(&libraryResolver{t: t})
			typeResolver.SetRequiredFields(tc.mode)

			got := map[string]bool{}
			for _, ctrl := range springmvc.NewParser(typeResolver).ExtractControllers(results) {
				for _, ep := range ctrl.Endpoints {
					if ep.Path != "/orders/search" {
						continue
					}
					for _, param := range ep.Parameters {
						if param.ParamType != "query" {
							t.Errorf("%s is bound from %s, want query", param.Name, param.ParamType)
						}
						got[param.Name] = param.Required
					}
				}
			}
			if len(got) != len(tc.want) {
				t.Fatalf("parameters = %v, want %v", got, tc.want)
			}
			for name, required := range tc.want {
				if got[name] != required {
					t.Errorf("%s required = %v, want %v", name, got[name], required)
				}
			}
			if unresolved := typeResolver.Unresolved(); len(unresolved) != 0 {
				t.Errorf("unresolved = %v", unresolved)
			}
		})
	}
}

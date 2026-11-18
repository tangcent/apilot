package parser

import (
	"reflect"
	"testing"
)

// A Kotlin enum decompiled by CFR is a class that extends java.lang.Enum, with
// one `/* enum */` field per constant, not an `enum` declaration. The sources
// below are the shapes CFR gives such classes.
func TestParser_ReadsAnEnumCFRPrintedAsAClass(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   []string
	}{
		{
			// Constants initialised at their declaration, with the synthetic
			// $VALUES and, from Kotlin 1.9, $ENTRIES.
			name: "initialised constants",
			source: `/*
 * Decompiled with CFR 0.152.
 */
package com.example.lib;

import kotlin.Metadata;
import kotlin.enums.EnumEntries;
import kotlin.enums.EnumEntriesKt;
import org.jetbrains.annotations.NotNull;

@Metadata(mv={1, 9, 0}, k=1, xi=48)
public final class OrderState
extends Enum<OrderState> {
    public static final /* enum */ OrderState PENDING = new OrderState("p");
    public static final /* enum */ OrderState PAID = new OrderState("d");
    public static final /* enum */ OrderState CLOSED = new OrderState("c");
    private static final /* synthetic */ OrderState[] $VALUES;
    private static final /* synthetic */ EnumEntries $ENTRIES;
    @NotNull
    private final String code;

    private OrderState(String code) {
        this.code = code;
    }

    @NotNull
    public final String getCode() {
        return this.code;
    }

    public static OrderState[] values() {
        return (OrderState[])$VALUES.clone();
    }

    public static OrderState valueOf(String value) {
        return Enum.valueOf(OrderState.class, value);
    }

    @NotNull
    public static EnumEntries<OrderState> getEntries() {
        return $ENTRIES;
    }

    static {
        $VALUES = orderStateArray = new OrderState[]{OrderState.PENDING, OrderState.PAID, OrderState.CLOSED};
        $ENTRIES = EnumEntriesKt.enumEntries((Enum[])$VALUES);
    }
}
`,
			want: []string{"PENDING", "PAID", "CLOSED"},
		},
		{
			// An enum with a companion object: the constants are assigned in the
			// class initializer, so their fields have no initializer, and the
			// companion's field is a static of the enum's own type with no marker.
			name: "constants assigned in the class initializer",
			source: `package com.example.lib;

import kotlin.Metadata;
import kotlin.jvm.JvmStatic;
import org.jetbrains.annotations.NotNull;

public final class Switch
extends Enum<Switch> {
    @NotNull
    public static final Companion Companion;
    private final int intValue;
    public static final /* enum */ Switch ON;
    public static final /* enum */ Switch OFF;
    private static final Switch FALLBACK;
    private static final /* synthetic */ Switch[] $VALUES;

    private Switch(int intValue) {
        this.intValue = intValue;
    }

    public final int getIntValue() {
        return this.intValue;
    }

    static {
        ON = new Switch(1);
        OFF = new Switch(0);
        $VALUES = switchArray = new Switch[]{Switch.ON, Switch.OFF};
        Companion = new Companion(null);
        FALLBACK = OFF;
    }

    public static final class Companion {
        private Companion() {
        }
    }
}
`,
			want: []string{"ON", "OFF"},
		},
		{
			// Constants with a body are nested classes of the enum, which is
			// then abstract.
			name: "constants with a body",
			source: `package com.example.lib;

import kotlin.Metadata;

public abstract class Mask
extends Enum<Mask> {
    public static final /* enum */ Mask PHONE = new PHONE("PHONE", 0);
    public static final /* enum */ Mask EMAIL = new EMAIL("EMAIL", 1);
    private static final /* synthetic */ Mask[] $VALUES;

    private Mask() {
    }

    public abstract String apply(String var1);

    static {
        $VALUES = maskArray = new Mask[]{Mask.PHONE, Mask.EMAIL};
    }

    static final class PHONE
    extends Mask {
        PHONE() {
        }

        public String apply(String value) {
            return value;
        }
    }

    static final class EMAIL
    extends Mask {
        EMAIL() {
        }

        public String apply(String value) {
            return value;
        }
    }
}
`,
			want: []string{"PHONE", "EMAIL"},
		},
		{
			name: "qualified superclass",
			source: `package com.example.lib;

public final class Level
extends java.lang.Enum<Level> {
    public static final /* enum */ Level LOW = new Level();
    public static final /* enum */ Level HIGH = new Level();
    private static final /* synthetic */ Level[] $VALUES;
}
`,
			want: []string{"LOW", "HIGH"},
		},
		{
			name: "no constants",
			source: `package com.example.lib;

public final class Nothing
extends Enum<Nothing> {
    private static final /* synthetic */ Nothing[] $VALUES;
}
`,
			want: nil,
		},
	}

	p, err := NewParser(ParserOptions{LogLevel: LogLevelError})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			classes, err := p.ParseSource([]byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			// The declaration comes first; the nested classes of a constant
			// with a body follow it.
			enum := classes[0]
			if !enum.IsEnum {
				t.Fatalf("%s is not an enum: %+v", enum.Name, enum)
			}
			if !reflect.DeepEqual(enum.EnumConstants, tt.want) {
				t.Errorf("constants = %v, want %v", enum.EnumConstants, tt.want)
			}
			if len(enum.EnumConstantDocs) != len(enum.EnumConstants) {
				t.Errorf("docs = %q, want one per constant", enum.EnumConstantDocs)
			}
			for _, nested := range classes[1:] {
				if nested.IsEnum {
					t.Errorf("%s, a class nested in the enum, is read as an enum", nested.Name)
				}
			}
		})
	}
}

// Only a class that extends java.lang.Enum is an enum: a class that merely
// looks like one, or one that extends another class named Enum-something, is
// not.
func TestParser_OnlyAClassExtendingEnumIsAnEnum(t *testing.T) {
	p, err := NewParser(ParserOptions{LogLevel: LogLevelError})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	classes, err := p.ParseSource([]byte(`package com.example;

public class Plain {
    public static final /* enum */ Plain ONE = new Plain();
    public static final Plain TWO = new Plain();
}

class Derived extends EnumBase<Derived> {
}

class Qualified extends com.example.Enum<Qualified> {
}

class Implementing implements Comparable<Implementing> {
}

class Standalone {
}

enum Declared {
    A, B
}
`))
	if err != nil {
		t.Fatal(err)
	}
	isEnum := map[string]bool{}
	for _, c := range classes {
		isEnum[c.Name] = c.IsEnum
	}
	want := map[string]bool{
		"Plain": false, "Derived": false, "Qualified": false, "Implementing": false, "Standalone": false,
		"Declared": true,
	}
	if !reflect.DeepEqual(isEnum, want) {
		t.Errorf("IsEnum by class = %v, want %v", isEnum, want)
	}
	for _, c := range classes {
		if !c.IsEnum && len(c.EnumConstants) != 0 {
			t.Errorf("%s is not an enum but has constants %v", c.Name, c.EnumConstants)
		}
	}
}

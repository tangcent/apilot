package parser

import "testing"

// An enum records whether it declares @JsonValue, which changes the value
// Jackson writes for it, however the enum is printed: as an enum declaration,
// or as the class CFR prints a Kotlin enum as.
func TestParser_EnumRecordsWhetherItDeclaresJsonValue(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   bool
	}{
		{
			name: "no annotation",
			source: `package a;
enum Level {
    LOW("l");
    private final String code;
    Level(String code) { this.code = code; }
    public String getCode() { return code; }
}`,
		},
		{
			name: "on a method",
			source: `package a;
import com.fasterxml.jackson.annotation.JsonValue;
enum Level {
    LOW("l");
    private final String code;
    Level(String code) { this.code = code; }
    @JsonValue
    public String getCode() { return code; }
}`,
			want: true,
		},
		{
			name: "on a field",
			source: `package a;
import com.fasterxml.jackson.annotation.JsonValue;
enum Level {
    LOW("l");
    @JsonValue
    private final String code;
    Level(String code) { this.code = code; }
}`,
			want: true,
		},
		{
			name: "by its qualified name",
			source: `package a;
enum Level {
    LOW;
    @com.fasterxml.jackson.annotation.JsonValue
    public String toValue() { return "l"; }
}`,
			want: true,
		},
		{
			name: "switched off",
			source: `package a;
import com.fasterxml.jackson.annotation.JsonValue;
enum Level {
    LOW;
    @JsonValue(false)
    public String toValue() { return "l"; }
}`,
		},
		{
			name: "only inside the body of a constant",
			source: `package a;
import com.fasterxml.jackson.annotation.JsonValue;
enum Level {
    LOW {
        @JsonValue
        public String toValue() { return "l"; }
    };
}`,
		},
		{
			name: "an enum without a body",
			source: `package a;
enum Level { LOW, HIGH }`,
		},
		{
			name: "decompiled from Kotlin, on a getter",
			source: `package com.example.lib;

import com.fasterxml.jackson.annotation.JsonValue;
import org.jetbrains.annotations.NotNull;

public final class Tier
extends Enum<Tier> {
    public static final /* enum */ Tier GOLD = new Tier("gold");
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
}`,
			want: true,
		},
		{
			name: "decompiled from Kotlin, without the annotation",
			source: `package com.example.lib;

import org.jetbrains.annotations.NotNull;

public final class Region
extends Enum<Region> {
    public static final /* enum */ Region NORTH = new Region("n");
    @NotNull
    private final String code;

    private Region(String code) {
        this.code = code;
    }

    @NotNull
    public final String getCode() {
        return this.code;
    }
}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewParser(ParserOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer p.Close()
			classes, err := p.ParseSource([]byte(tt.source))
			if err != nil {
				t.Fatal(err)
			}
			if len(classes) != 1 || !classes[0].IsEnum {
				t.Fatalf("classes = %+v, want one enum", classes)
			}
			if got := classes[0].EnumJsonValue; got != tt.want {
				t.Errorf("EnumJsonValue = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasJsonValue(t *testing.T) {
	tests := []struct {
		name        string
		annotations []Annotation
		want        bool
	}{
		{name: "none"},
		{name: "plain", annotations: []Annotation{{Name: "JsonValue"}}, want: true},
		{name: "qualified", annotations: []Annotation{{Name: "com.fasterxml.jackson.annotation.JsonValue"}}, want: true},
		{name: "switched on", annotations: []Annotation{{Name: "JsonValue", Params: map[string]string{"value": "true"}}}, want: true},
		{name: "switched off", annotations: []Annotation{{Name: "JsonValue", Params: map[string]string{"value": "false"}}}},
		{name: "another annotation", annotations: []Annotation{{Name: "JsonProperty"}, {Name: "NotNull"}}},
		{name: "among others", annotations: []Annotation{{Name: "NotNull"}, {Name: "JsonValue"}}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasJsonValue(tt.annotations); got != tt.want {
				t.Errorf("HasJsonValue = %v, want %v", got, tt.want)
			}
		})
	}
}

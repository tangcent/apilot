package parser

import (
	"reflect"
	"testing"
)

func parseEnumSource(t *testing.T, source string) Class {
	t.Helper()
	jp, err := NewParser(ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer jp.Close()
	classes, err := jp.ParseSource([]byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if len(classes) == 0 || !classes[0].IsEnum {
		t.Fatalf("classes = %+v, want an enum first", classes)
	}
	return classes[0]
}

// An enum that CFR printed as a class records whether a constant is given a name by
// @JsonProperty, as one declared as an enum does. CFR prints the annotations of a
// constant before its field, among the modifiers, and marks the field with a comment.
func TestParser_DecompiledEnumRecordsWhetherAConstantIsRenamed(t *testing.T) {
	tests := []struct {
		name      string
		constants string
		renamed   bool
	}{
		{
			name: "no annotation",
			constants: `    public static final /* enum */ Level LOW = new Level();
    public static final /* enum */ Level HIGH = new Level();`,
		},
		{
			name: "a name on a constant",
			constants: `    public static final /* enum */ Level LOW = new Level();
    @JsonProperty(value="high_x")
    public static final /* enum */ Level HIGH = new Level();`,
			renamed: true,
		},
		{
			name: "a name given as the only element",
			constants: `    @JsonProperty("low_x")
    public static final /* enum */ Level LOW = new Level();
    public static final /* enum */ Level HIGH = new Level();`,
			renamed: true,
		},
		{
			name: "a name among other annotations",
			constants: `    @Deprecated
    @JsonProperty(value="low_x")
    @NotNull
    public static final /* enum */ Level LOW = new Level();
    public static final /* enum */ Level HIGH = new Level();`,
			renamed: true,
		},
		{
			name: "a qualified annotation",
			constants: `    @com.fasterxml.jackson.annotation.JsonProperty(value="low_x")
    public static final /* enum */ Level LOW = new Level();
    public static final /* enum */ Level HIGH = new Level();`,
			renamed: true,
		},
		{
			name: "a constant that CFR left without an initializer",
			constants: `    @JsonProperty(value="low_x")
    public static final /* enum */ Level LOW;
    public static final /* enum */ Level HIGH;`,
			renamed: true,
		},
		{
			name: "an empty name gives none",
			constants: `    @JsonProperty(value="")
    public static final /* enum */ Level LOW = new Level();
    public static final /* enum */ Level HIGH = new Level();`,
		},
		{
			name: "no name gives none",
			constants: `    @JsonProperty
    public static final /* enum */ Level LOW = new Level();
    public static final /* enum */ Level HIGH = new Level();`,
		},
		{
			name: "another annotation",
			constants: `    @Deprecated
    public static final /* enum */ Level LOW = new Level();
    public static final /* enum */ Level HIGH = new Level();`,
		},
		{
			name: "a name on a static field that is no constant",
			constants: `    public static final /* enum */ Level LOW = new Level();
    public static final /* enum */ Level HIGH = new Level();
    @JsonProperty(value="default_x")
    public static final Level DEFAULT = Level.LOW;`,
		},
		{
			name: "a name on a member that is no field",
			constants: `    public static final /* enum */ Level LOW = new Level();
    public static final /* enum */ Level HIGH = new Level();
    @JsonProperty(value="code_x")
    public final String getCode() {
        return null;
    }`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := parseEnumSource(t, `/*
 * Decompiled with CFR 0.152.
 */
package com.example.lib;

import com.fasterxml.jackson.annotation.JsonProperty;
import kotlin.Metadata;
import org.jetbrains.annotations.NotNull;

@Metadata(mv={1, 9, 0}, k=1, xi=48)
public final class Level
extends Enum<Level> {
`+tt.constants+`
    private static final /* synthetic */ Level[] $VALUES;

    private Level() {
    }
}
`)
			if level.EnumJsonProperty != tt.renamed {
				t.Errorf("EnumJsonProperty = %v, want %v", level.EnumJsonProperty, tt.renamed)
			}
			if want := []string{"LOW", "HIGH"}; !reflect.DeepEqual(level.EnumConstants, want) {
				t.Errorf("constants = %q, want %q", level.EnumConstants, want)
			}
		})
	}
}

// An enum records the same facts about how Jackson writes it whether it is declared
// as an enum or printed by CFR as a class that extends Enum: the interfaces it
// implements, the annotations of the enum, whether a member carries @JsonValue and
// whether a constant is given a name by @JsonProperty.
func TestParser_AnEnumPrintedAsAClassRecordsTheFactsOfItsDeclaration(t *testing.T) {
	tests := []struct {
		name        string
		declaration string // an enum Level, as it is written
		decompiled  string // the same enum, as CFR prints it from Kotlin
		recorded    func(Class) bool
	}{
		{
			name:        "interfaces",
			recorded:    func(c Class) bool { return len(c.Interfaces) == 2 },
			declaration: `enum Level implements Coded, Other<String> { LOW }`,
			decompiled: `public final class Level
extends Enum<Level>
implements Coded,
Other<String> {
    public static final /* enum */ Level LOW = new Level();
}`,
		},
		{
			name:        "the annotations of the enum",
			recorded:    func(c Class) bool { return len(c.Annotations) == 2 },
			declaration: `@JsonFormat(shape=JsonFormat.Shape.OBJECT) @JsonSerialize(using=Writer.class) enum Level { LOW }`,
			decompiled: `@JsonFormat(shape=JsonFormat.Shape.OBJECT)
@JsonSerialize(using=Writer.class)
public final class Level
extends Enum<Level> {
    public static final /* enum */ Level LOW = new Level();
}`,
		},
		{
			name:        "@JsonValue on a method",
			recorded:    func(c Class) bool { return c.EnumJsonValue },
			declaration: `enum Level { LOW; @JsonValue public String code() { return null; } }`,
			decompiled: `public final class Level
extends Enum<Level> {
    public static final /* enum */ Level LOW = new Level();

    @JsonValue
    public final String code() {
        return null;
    }
}`,
		},
		{
			name:        "@JsonValue on a field",
			recorded:    func(c Class) bool { return c.EnumJsonValue },
			declaration: `enum Level { LOW; @JsonValue private final String code = null; }`,
			decompiled: `public final class Level
extends Enum<Level> {
    public static final /* enum */ Level LOW = new Level();
    @JsonValue
    private final String code = null;
}`,
		},
		{
			name:        "a name on a constant",
			recorded:    func(c Class) bool { return c.EnumJsonProperty },
			declaration: `enum Level { @JsonProperty("low_x") LOW, HIGH }`,
			decompiled: `public final class Level
extends Enum<Level> {
    @JsonProperty(value="low_x")
    public static final /* enum */ Level LOW = new Level();
    public static final /* enum */ Level HIGH = new Level();
}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const imports = "package a;\nimport com.fasterxml.jackson.annotation.*;\n"
			declared := parseEnumSource(t, imports+tt.declaration)
			printed := parseEnumSource(t, imports+tt.decompiled)
			if !tt.recorded(declared) {
				t.Fatalf("the declaration records no such fact: %+v", declared)
			}

			if !reflect.DeepEqual(printed.Interfaces, declared.Interfaces) {
				t.Errorf("interfaces = %q, want %q", printed.Interfaces, declared.Interfaces)
			}
			if !reflect.DeepEqual(printed.Annotations, declared.Annotations) {
				t.Errorf("annotations = %+v, want %+v", printed.Annotations, declared.Annotations)
			}
			if printed.EnumJsonValue != declared.EnumJsonValue {
				t.Errorf("EnumJsonValue = %v, want %v", printed.EnumJsonValue, declared.EnumJsonValue)
			}
			if printed.EnumJsonProperty != declared.EnumJsonProperty {
				t.Errorf("EnumJsonProperty = %v, want %v", printed.EnumJsonProperty, declared.EnumJsonProperty)
			}
		})
	}
}

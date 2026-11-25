package parser

import (
	"reflect"
	"testing"
)

// An enum records whether an entry is given a name by @JsonProperty, as the Java
// parser does for a constant, and the interfaces it implements. Of the use-site
// targets only the entry and its field count.
func TestEnumRecordsWhetherAnEntryIsRenamed(t *testing.T) {
	tests := []struct {
		name    string
		entries string
		renamed bool
	}{
		{name: "no annotation", entries: "LOW,\n    HIGH,"},
		{name: "a name on an entry", entries: "LOW,\n    @JsonProperty(\"high_x\") HIGH,", renamed: true},
		{name: "a name on the field of an entry", entries: "@field:JsonProperty(value = \"mid_x\") @Deprecated(\"x\") MID,", renamed: true},
		{name: "a name on the getter of an entry", entries: "@get:JsonProperty(\"never\") NEVER,"},
		{name: "an empty name gives none", entries: "@JsonProperty(\"\") NONE,"},
		{name: "another annotation", entries: "@Deprecated(\"x\") LOW,"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSource(t, "src/Level.kt", "package p\n\nenum class Level : Coded {\n    "+tt.entries+"\n}\n")
			level := result.Classes[0]
			if level.EnumJsonProperty != tt.renamed {
				t.Errorf("EnumJsonProperty = %v, want %v", level.EnumJsonProperty, tt.renamed)
			}
			if len(level.EnumValues) == 0 {
				t.Errorf("entries = %q, want the entries read", level.EnumValues)
			}
			if want := []string{"Coded"}; !reflect.DeepEqual(level.Interfaces, want) {
				t.Errorf("interfaces = %q, want %q", level.Interfaces, want)
			}
		})
	}
}

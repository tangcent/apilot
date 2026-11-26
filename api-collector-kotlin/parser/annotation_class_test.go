package parser

import (
	"reflect"
	"testing"
)

// The grammar reads `annotation class` as an infix expression, and the
// annotations in front of it would otherwise stick to the next declaration.
// A primary constructor makes the grammar read it as a class already
// (`annotation class X(val v: Int)`); a body `{ }` and a parameter default
// of `[]` do not, and are read back here. Each form is the annotation class,
// the declaration after it keeps only its own annotations, and a class the
// grammar already reads is not rewritten.
func TestAnnotationClassesAreReadWithoutStickingToTheNextDeclaration(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		ann       string
		annAnns   []string
		next      string
		nextAnns  []string
		grammarOK bool
	}{
		{
			name: "no parameters",
			source: `package p

@JsonValue
annotation class AsValue

@Deprecated("own")
class Next
`,
			ann: "AsValue", annAnns: []string{"JsonValue"},
			next: "Next", nextAnns: []string{"Deprecated"},
		},
		{
			name: "a constructor parameter",
			source: `package p

@JsonValue
annotation class AsValue(val v: Int)

@Deprecated("own")
class Next
`,
			ann: "AsValue", annAnns: []string{"JsonValue"},
			next: "Next", nextAnns: []string{"Deprecated"},
		},
		{
			name: "vararg and an array default",
			source: `package p

@Constraint(validatedBy = [])
annotation class Sized(
    vararg val value: Int = [],
    val groups: Array<KClass<*>> = [],
)

@Deprecated("own")
class Next
`,
			ann: "Sized", annAnns: []string{"Constraint"},
			next: "Next", nextAnns: []string{"Deprecated"},
		},
		{
			name: "a class body",
			source: `package p

@JsonValue
annotation class AsValue {
}

@Deprecated("own")
class Next
`,
			ann: "AsValue", annAnns: []string{"JsonValue"},
			next: "Next", nextAnns: []string{"Deprecated"},
		},
		{
			name: "internal",
			source: `package p

@JsonValue
internal annotation class AsValue

@Deprecated("own")
class Next
`,
			ann: "AsValue", annAnns: []string{"JsonValue"},
			next: "Next", nextAnns: []string{"Deprecated"},
		},
		{
			name: "public",
			source: `package p

@JsonValue
public annotation class AsValue

@Deprecated("own")
class Next
`,
			ann: "AsValue", annAnns: []string{"JsonValue"},
			next: "Next", nextAnns: []string{"Deprecated"},
		},
		{
			name: "several annotations",
			source: `package p

@Target(AnnotationTarget.FUNCTION)
@Retention(AnnotationRetention.RUNTIME)
@MustBeDocumented
@Constraint(validatedBy = [])
annotation class Sized(val v: Int)

@Deprecated("own")
data class Next(val v: Int)
`,
			ann: "Sized", annAnns: []string{"Target", "Retention", "MustBeDocumented", "Constraint"},
			next: "Next", nextAnns: []string{"Deprecated"},
		},
		{
			name: "nested in a class",
			source: `package p

class Outer {
    @JsonValue
    annotation class AsValue

    @Deprecated("own")
    class Next
}
`,
			ann: "AsValue", annAnns: []string{"JsonValue"},
			next: "Next", nextAnns: []string{"Deprecated"},
		},
		{
			name: "a class the grammar already reads",
			source: `package p

@Deprecated("own")
class Next
`,
			next: "Next", nextAnns: []string{"Deprecated"},
			grammarOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSource(t, "src/Ann.kt", tt.source)
			if !tt.grammarOK {
				ann := classNamed(t, result, tt.ann)
				if !ann.IsAnnotation {
					t.Errorf("%s IsAnnotation = false", tt.ann)
				}
				if got := annotationNames(ann.Annotations); !reflect.DeepEqual(got, tt.annAnns) {
					t.Errorf("%s annotations = %v, want %v", tt.ann, got, tt.annAnns)
				}
			}
			next := classNamed(t, result, tt.next)
			if next.IsAnnotation {
				t.Errorf("%s was read as an annotation class", tt.next)
			}
			if got := annotationNames(next.Annotations); !reflect.DeepEqual(got, tt.nextAnns) {
				t.Errorf("%s annotations = %v, want %v", tt.next, got, tt.nextAnns)
			}
		})
	}
}

func classNamed(t *testing.T, result *ParseResult, name string) Class {
	t.Helper()
	for _, class := range result.Classes {
		if class.Name == name {
			return class
		}
	}
	t.Fatalf("no class %s in %v", name, classNames(result))
	return Class{}
}

func classNames(result *ParseResult) []string {
	var names []string
	for _, class := range result.Classes {
		names = append(names, class.Name)
	}
	return names
}

func annotationNames(annotations []Annotation) []string {
	var names []string
	for _, ann := range annotations {
		names = append(names, ann.Name)
	}
	return names
}

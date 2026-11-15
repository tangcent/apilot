package doc

import (
	"strings"

	"github.com/tangcent/apilot/api-collector-java/parser"
)

// Deprecation is what a declaration's source says about its deprecation.
type Deprecation struct {
	Deprecated bool
	// Note is the message of Kotlin's @Deprecated or the text of a
	// @deprecated doc tag. It is empty when the source gives none, as Java's
	// @Deprecated never does.
	Note string
}

// DeprecationOf reads the deprecation of one declaration from its annotations
// and its doc comment as written (Javadoc or KDoc, markers and tags
// included): a @Deprecated annotation, Java's or Kotlin's, or a @deprecated
// block tag. The annotation's message wins over the tag's text.
func DeprecationOf(annotations []parser.Annotation, rawDoc string) Deprecation {
	var d Deprecation
	for _, ann := range annotations {
		if simpleName(ann.Name) != "Deprecated" {
			continue
		}
		d.Deprecated = true
		d.Note = firstNonEmpty(ann.Params["message"], ann.Params["value"])
	}
	if text, ok := DocTag(rawDoc, "deprecated"); ok {
		d.Deprecated = true
		d.Note = firstNonEmpty(d.Note, text)
	}
	return d
}

// EndpointDeprecation is the deprecation of a handler method, else of the
// classes it belongs to, nearest first: every endpoint of a deprecated class
// is deprecated.
func EndpointDeprecation(method Deprecation, classes ...Deprecation) Deprecation {
	if method.Deprecated {
		return method
	}
	for _, c := range classes {
		if c.Deprecated {
			return c
		}
	}
	return Deprecation{}
}

// DocTag returns the text of the first block tag called name in a doc comment
// as written, its continuation lines included, and whether the comment has
// such a tag.
func DocTag(rawDoc, name string) (string, bool) {
	tag := "@" + name
	var text []string
	found := false
	for _, line := range strings.Split(rawDoc, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(strings.TrimPrefix(line, "/**"), "*/")
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*"))
		if found {
			if line == "" || strings.HasPrefix(line, "@") {
				break
			}
			text = append(text, line)
			continue
		}
		if line == tag || strings.HasPrefix(line, tag+" ") || strings.HasPrefix(line, tag+"\t") {
			found = true
			if rest := strings.TrimSpace(strings.TrimPrefix(line, tag)); rest != "" {
				text = append(text, rest)
			}
		}
	}
	return strings.Join(text, "\n"), found
}

func simpleName(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}

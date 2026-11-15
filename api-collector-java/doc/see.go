package doc

import "strings"

// SeeReferences returns the program elements the @see tags of a doc comment
// name, in order: the first word of each tag, such as "OrderState",
// "com.example.OrderState" or "OrderState#code". A KDoc link in brackets,
// "[OrderState]", is unwrapped. A tag that quotes a string or links a URL
// names no program element and is left out.
func SeeReferences(rawDoc string) []string {
	var refs []string
	for _, line := range strings.Split(rawDoc, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(strings.TrimPrefix(line, "/**"), "*/")
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "*"))
		if !strings.HasPrefix(line, "@see ") && !strings.HasPrefix(line, "@see\t") {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "@see"))
		if len(fields) == 0 {
			continue
		}
		ref := fields[0]
		if strings.HasPrefix(ref, "[") && strings.HasSuffix(ref, "]") {
			ref = strings.TrimSuffix(strings.TrimPrefix(ref, "["), "]")
		}
		if ref == "" || strings.HasPrefix(ref, "\"") || strings.HasPrefix(ref, "<") {
			continue
		}
		refs = append(refs, ref)
	}
	return refs
}

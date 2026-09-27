// Package jsdoc normalizes JavaScript/TypeScript comment text (JSDoc block
// comments, line comments, or plain description strings) into the shared
// docmeta contract.
package jsdoc

import (
	"strings"

	docmeta "github.com/tangcent/apilot/api-docmeta"
)

// Extract parses a raw comment and returns its documented metadata. The
// lines before the first block tag form the description; recognized tags are
// @example (demo value) and @default / @defaultValue (documented default).
// Text that is not comment syntax is treated as a plain description.
func Extract(raw string) docmeta.Documentation {
	var d docmeta.Documentation
	var description []string
	for _, line := range commentLines(raw) {
		if strings.HasPrefix(line, "@") {
			tag, value := splitTagLine(line)
			switch tag {
			case "@example":
				d.Demo = value
			case "@default", "@defaultValue":
				d.DefaultValue = value
			}
			continue
		}
		description = append(description, line)
	}
	d.Comment = strings.Join(description, " ")
	return d
}

func commentLines(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	switch {
	case strings.HasPrefix(raw, "//"):
		var lines []string
		for _, line := range strings.Split(raw, "\n") {
			line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "//"))
			if line != "" {
				lines = append(lines, line)
			}
		}
		return lines
	case strings.HasPrefix(raw, "/*"):
		body := strings.TrimPrefix(raw, "/*")
		body = strings.TrimSuffix(body, "*/")
		var lines []string
		for _, line := range strings.Split(body, "\n") {
			line = strings.TrimSpace(line)
			line = strings.TrimSpace(strings.TrimPrefix(line, "*"))
			if line != "" {
				lines = append(lines, line)
			}
		}
		return lines
	default:
		return []string{raw}
	}
}

func splitTagLine(line string) (tag string, value string) {
	parts := strings.SplitN(line, " ", 2)
	tag = parts[0]
	if len(parts) == 2 {
		value = strings.TrimSpace(parts[1])
	}
	return tag, value
}

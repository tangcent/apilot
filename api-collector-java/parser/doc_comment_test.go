package parser

import "testing"

func TestCleanDocComment(t *testing.T) {
	for _, tc := range []struct {
		name        string
		raw         string
		includeTags bool
		want        string
	}{
		{"a wrapped line stays a line",
			"/**\n * First line\n * goes on.\n */", false,
			"First line\ngoes on."},
		{"a blank line keeps paragraphs apart",
			"/**\n * Title\n *\n * Detail.\n */", false,
			"Title\n\nDetail."},
		{"several blank lines are one",
			"/**\n * Title\n *\n *\n * Detail.\n */", false,
			"Title\n\nDetail."},
		{"blank lines at either end are dropped",
			"/**\n *\n * Title\n *\n */", false,
			"Title"},
		{"the description ends at the first block tag",
			"/**\n * Title\n *\n * @param id the id\n *     continued\n * @return the user\n */", false,
			"Title"},
		{"with tags, every line is kept",
			"/**\n * Title\n *\n * @param id the id\n * @return the user\n */", true,
			"Title\n\n@param id the id\n@return the user"},
		{"a one-line comment",
			"/** Title. */", false,
			"Title."},
		{"no comment",
			"", false,
			""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := CleanDocComment(tc.raw, tc.includeTags); got != tc.want {
				t.Errorf("CleanDocComment(%q, %v) = %q, want %q", tc.raw, tc.includeTags, got, tc.want)
			}
		})
	}
}

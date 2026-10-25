package parser

import "testing"

func TestMediaTypes(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "wire media type passes through",
			raw:  "application/json",
			want: []string{"application/json"},
		},
		{
			name: "quoted literal is unquoted",
			raw:  `"multipart/form-data"`,
			want: []string{"multipart/form-data"},
		},
		{
			name: "constant reference resolves to the wire type",
			raw:  "MediaType.MULTIPART_FORM_DATA",
			want: []string{"multipart/form-data"},
		},
		{
			name: "fully qualified constant reference resolves",
			raw:  "jakarta.ws.rs.core.MediaType.APPLICATION_JSON",
			want: []string{"application/json"},
		},
		{
			name: "array literal is split per entry",
			raw:  "application/json,text/plain",
			want: []string{"application/json", "text/plain"},
		},
		{
			name: "array of constants resolves each entry",
			raw:  "MediaType.APPLICATION_JSON,MediaType.TEXT_PLAIN",
			want: []string{"application/json", "text/plain"},
		},
		{
			name: "empty value declares nothing",
			raw:  "",
			want: nil,
		},
		{
			name: "unknown constant is kept as written",
			raw:  "MediaType.CUSTOM_TYPE",
			want: []string{"MediaType.CUSTOM_TYPE"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := MediaTypes(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("MediaTypes(%q) = %v, want %v", tt.raw, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("MediaTypes(%q)[%d] = %q, want %q", tt.raw, i, got[i], tt.want[i])
				}
			}
		})
	}
}

package main

import (
	"reflect"
	"testing"
)

func TestParseModelNames(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want []string
	}{
		{
			name: "plain list output",
			out:  "Available models:\n\nmodels without spaces are ignored\n" + "\tgemini-3-flash-preview\n gemini-3-pro\n",
			want: []string{"gemini-3-flash-preview", "gemini-3-pro"},
		},
		{
			name: "models/ prefix is stripped",
			out:  "models/gemini-3-flash\n",
			want: []string{"gemini-3-flash"},
		},
		{
			name: "header and upgrade lines are skipped",
			out:  "Available models:\nUpgrade available: 1.2.3\n",
			want: nil,
		},
		{
			name: "empty output",
			out:  "",
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseModelNames(tt.out)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseModelNames(%q) = %v, want %v", tt.out, got, tt.want)
			}
		})
	}
}

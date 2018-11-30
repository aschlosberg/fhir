package jsonfhir

import "testing"

func TestUpperCaseFirst(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{
			in:   "helloWorld",
			want: "HelloWorld",
		},
		{
			in:   "HelloWorld",
			want: "HelloWorld",
		},
		{
			in:   "3lloWorld",
			want: "3lloWorld",
		},
		{
			in:   "",
			want: "",
		},
		{
			in:   "X",
			want: "X",
		},
		{
			in:   "y",
			want: "Y",
		},
		{
			in:   "0",
			want: "0",
		},
	}

	for _, tt := range tests {
		if got, want := upperCaseFirst(tt.in), tt.want; got != want {
			t.Errorf("upperCaseFirst(%q) got %q; want %q", tt.in, got, want)
		}
	}
}

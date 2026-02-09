package middleware

import "testing"

func TestNewOriginMatcher(t *testing.T) {
	matcher := NewOriginMatcher("http://localhost:5173,https://*.pulse.app")

	cases := []struct {
		origin string
		want   bool
	}{
		{origin: "http://localhost:5173", want: true},
		{origin: "https://api.pulse.app", want: true},
		{origin: "https://foo.bar.pulse.app", want: true},
		{origin: "https://pulse.app", want: false},
		{origin: "http://localhost:5174", want: false},
		{origin: "not-an-origin", want: false},
	}

	for _, tc := range cases {
		got := matcher(tc.origin)
		if got != tc.want {
			t.Fatalf("matcher(%q) = %v, want %v", tc.origin, got, tc.want)
		}
	}
}

func TestNewOriginMatcherAllowAll(t *testing.T) {
	matcher := NewOriginMatcher("*")
	if !matcher("https://anywhere.example") {
		t.Fatal("expected wildcard matcher to allow any origin")
	}
}

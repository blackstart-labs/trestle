package identity

import "testing"

func TestPublicOrigin(t *testing.T) {
	for raw, want := range map[string]string{
		"":                               "",
		" https://Trestle.Example:443/ ": "https://trestle.example",
		"https://trestle.example:8443":   "https://trestle.example:8443",
	} {
		got, err := publicOrigin(raw)
		if got != want || err != nil {
			t.Errorf("publicOrigin(%q) = %q, %v; want %q", raw, got, err, want)
		}
	}
	for _, raw := range []string{
		"http://trestle.example", "https://user@trestle.example", "https://trestle.example/path",
		"https://trestle.example?", "https://trestle.example#", "https://*.example.com",
		"https://trestle.example:65536", "https://127.0.0.1", "https://127.0.0.0x1",
		"https://[trestle.example]", "https://trestle.example:",
	} {
		if got, err := publicOrigin(raw); got != "" || err != ErrInvalidPublicOrigin {
			t.Errorf("unexpected acceptance of %q: %q, %v", raw, got, err)
		}
	}
}

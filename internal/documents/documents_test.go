package documents

import (
	"strings"
	"testing"
)

func TestValidationAndLinks(t *testing.T) {
	for _, in := range []Input{{Title: ""}, {Title: strings.Repeat("界", 161)}, {Title: "Page", Content: strings.Repeat("x", 100001)}, {Title: "Page", Tags: []string{"a", "a"}}, {Title: "Page", Content: "a\x00b"}} {
		if Validate(in) == nil {
			t.Fatalf("accepted invalid input: %q", in.Title)
		}
	}
	if err := Validate(Input{Title: "বাংলা", Content: "  keep\n\nwhitespace", Tags: []string{"Research"}}); err != nil {
		t.Fatal(err)
	}
	links := Links("[[123e4567-e89b-12d3-a456-426614174000|Page]] [[123e4567-e89b-12d3-a456-426614174000]] [[bad]]")
	if len(links) != 1 {
		t.Fatalf("links: %v", links)
	}
}

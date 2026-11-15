package boards

import (
	"math"
	"strings"
	"testing"
)

func TestValidateScene(t *testing.T) {
	shape := Element{ID: "a", Kind: "rectangle", X: 10, Y: 10, Width: 100, Height: 50, Color: "ink"}
	for name, change := range map[string]func(*Scene){
		"unknown version":    func(s *Scene) { s.Version = 2 },
		"unknown shape":      func(s *Scene) { s.Elements[0].Kind = "iframe" },
		"duplicate id":       func(s *Scene) { s.Elements = append(s.Elements, shape) },
		"invalid coordinate": func(s *Scene) { s.Elements[0].X = math.NaN() },
		"out of bounds":      func(s *Scene) { s.Elements[0].X = 3999 },
		"raw color":          func(s *Scene) { s.Elements[0].Color = "url(https://example.com)" },
		"long label":         func(s *Scene) { s.Elements[0].Text = strings.Repeat("x", 201) },
		"dangling arrow": func(s *Scene) {
			s.Elements = append(s.Elements, Element{ID: "edge", Kind: "arrow", Color: "ink", From: "a", To: "missing"})
		},
		"too many objects": func(s *Scene) { s.Elements = make([]Element, 501) },
		"short stroke":     func(s *Scene) { s.Elements[0].Kind = "stroke"; s.Elements[0].Points = []Point{{1, 1}} },
		"points on shape":  func(s *Scene) { s.Elements[0].Points = []Point{{1, 1}, {2, 2}} },
		"unknown fill":     func(s *Scene) { s.Elements[0].Fill = "oklch(0.5 0.2 30)" },
		"thin line":        func(s *Scene) { s.Elements[0].StrokeWidth = 0.5 },
		"thick line":       func(s *Scene) { s.Elements[0].StrokeWidth = 9 },
		"tiny text":        func(s *Scene) { s.Elements[0].FontSize = 8 },
		"huge text":        func(s *Scene) { s.Elements[0].FontSize = 96 },
		"unknown align":    func(s *Scene) { s.Elements[0].Align = "justify" },
		"unknown route":    func(s *Scene) { s.Elements[0].Route = "bezier" },
		"unknown port":     func(s *Scene) { s.Elements[0].FromPort = "center" },
		"unknown heads":    func(s *Scene) { s.Elements[0].Heads = "start" },
		"long group":       func(s *Scene) { s.Elements[0].Group = strings.Repeat("g", 65) },
	} {
		t.Run(name, func(t *testing.T) {
			s := Scene{Version: 1, Elements: []Element{shape}}
			change(&s)
			if Validate("Board", s) == nil {
				t.Fatal("accepted invalid scene")
			}
		})
	}
	extended := Element{ID: "a", Kind: "database", X: 10, Y: 10, Width: 100, Height: 50, Color: "info", Fill: "info", StrokeWidth: 3, Dashed: true, FontSize: 24, Bold: true, Align: "left", Group: "g", Locked: true}
	if err := Validate("Board", Scene{Version: 1, Elements: []Element{extended}}); err != nil {
		t.Fatal(err)
	}
	if err := Validate("Board", Scene{Version: 1, Elements: []Element{shape}}); err != nil {
		t.Fatal(err)
	}
	if err := Validate("", Scene{Version: 1, Elements: []Element{}}); err == nil {
		t.Fatal("accepted empty title")
	}
}

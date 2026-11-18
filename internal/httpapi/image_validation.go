package httpapi

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strings"
)

// Never accept a claimed image MIME type as proof of image content. SVG is
// restricted to passive geometry; scripts, links, styling and external resources
// are rejected rather than served as same-origin active documents.
func validatedImageType(data []byte) (string, error) {
	if len(data) == 0 || len(data) > 5<<20 {
		return "", errors.New("Use a nonempty image no larger than 5 MB")
	}
	detected := http.DetectContentType(data)
	switch detected {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return detected, nil
	}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	depth := 0
	root := false
	closed := false
	tokens := 0
	elements := map[string]bool{"svg": true, "g": true, "path": true, "rect": true, "circle": true, "ellipse": true, "line": true, "polyline": true, "polygon": true, "title": true, "desc": true, "defs": true, "linearGradient": true, "radialGradient": true, "stop": true, "clipPath": true}
	attributes := map[string]bool{"xmlns": true, "viewBox": true, "width": true, "height": true, "x": true, "y": true, "x1": true, "x2": true, "y1": true, "y2": true, "cx": true, "cy": true, "r": true, "rx": true, "ry": true, "d": true, "points": true, "fill": true, "fill-rule": true, "fill-opacity": true, "stroke": true, "stroke-width": true, "stroke-linecap": true, "stroke-linejoin": true, "stroke-opacity": true, "stroke-dasharray": true, "opacity": true, "transform": true, "id": true, "offset": true, "stop-color": true, "stop-opacity": true, "gradientUnits": true, "gradientTransform": true, "preserveAspectRatio": true, "version": true}
	invalid := errors.New("Use PNG, JPEG, GIF, WebP, or a passive SVG without scripts, links, styles, or external resources")
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", invalid
		}
		tokens++
		if tokens > 50000 {
			return "", invalid
		}
		switch v := token.(type) {
		case xml.StartElement:
			if closed || !elements[v.Name.Local] || (v.Name.Space != "" && v.Name.Space != "http://www.w3.org/2000/svg") {
				return "", invalid
			}
			if !root {
				if v.Name.Local != "svg" {
					return "", invalid
				}
				root = true
			}
			depth++
			if depth > 64 {
				return "", invalid
			}
			for _, a := range v.Attr {
				if !attributes[a.Name.Local] || (a.Name.Space != "" && a.Name.Space != "xmlns") {
					return "", invalid
				}
				value := strings.ToLower(a.Value)
				if strings.Contains(value, "url(") || strings.Contains(value, "javascript:") || strings.ContainsAny(value, "\\") {
					return "", invalid
				}
				if a.Name.Local == "xmlns" && a.Value != "http://www.w3.org/2000/svg" {
					return "", invalid
				}
			}
		case xml.EndElement:
			depth--
			if depth == 0 {
				closed = true
			}
		case xml.Directive:
			return "", invalid
		case xml.ProcInst:
			if v.Target != "xml" || root {
				return "", invalid
			}
		case xml.CharData:
			if depth == 0 && strings.TrimSpace(string(v)) != "" {
				return "", invalid
			}
		}
	}
	if !root || !closed || depth != 0 {
		return "", invalid
	}
	return "image/svg+xml", nil
}

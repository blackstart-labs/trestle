package httpapi

import (
	"bytes"
	"encoding/base64"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestImageContentValidation(t *testing.T) {
	for _, raw := range []string{"not an image", `<html><script>alert(1)</script></html>`, `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"/>`, `<svg><script>alert(1)</script></svg>`, `<svg><image href="https://example.com"/></svg>`, `<svg><path style="fill:red"/></svg>`, `<!DOCTYPE svg><svg/>`, `<svg/><svg/>`, `<svg><path fill="url(https://example.com)"/></svg>`} {
		if _, err := validatedImageType([]byte(raw)); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	if kind, err := validatedImageType([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M0 0L24 24" stroke="black"/></svg>`)); err != nil || kind != "image/svg+xml" {
		t.Fatalf("safe SVG: %s %v", kind, err)
	}
	fake := "data:image/png;base64," + base64.StdEncoding.EncodeToString([]byte("pretend PNG"))
	if _, _, err := parseImageData(fake); err == nil {
		t.Fatal("trusted claimed MIME")
	}
	req := httptest.NewRequest("PUT", "/", strings.NewReader("not an image"))
	req.Header.Set("Content-Type", "image/png")
	if _, _, ok := extractUploadedImage(httptest.NewRecorder(), req); ok {
		t.Fatal("raw spoof accepted")
	}
}
func TestMultipartImageSizeBound(t *testing.T) {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	file, _ := writer.CreateFormFile("file", "huge.png")
	file.Write(bytes.Repeat([]byte{'x'}, 6<<20))
	writer.Close()
	req := httptest.NewRequest("PUT", "/", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if _, _, ok := extractUploadedImage(httptest.NewRecorder(), req); ok {
		t.Fatal("oversized upload accepted")
	}
}

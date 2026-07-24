package storage

import "testing"

func TestSplitDataURL(t *testing.T) {
	mediaType, body, err := splitDataURL("data:image/png;base64,aGVsbG8=")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mediaType != "image/png" {
		t.Errorf("mediaType = %q, want image/png", mediaType)
	}
	if string(body) != "hello" {
		t.Errorf("body = %q, want %q", body, "hello")
	}
}

func TestSplitDataURLRejectsNonDataURL(t *testing.T) {
	if _, _, err := splitDataURL("https://example.com/image.png"); err == nil {
		t.Fatal("expected error for non data: URL")
	}
}

func TestSplitDataURLRejectsNonBase64(t *testing.T) {
	if _, _, err := splitDataURL("data:image/png,notbase64"); err == nil {
		t.Fatal("expected error for non-base64 data URL")
	}
}

func TestExtForMediaType(t *testing.T) {
	if got := extForMediaType("image/png"); got != ".png" {
		t.Errorf("extForMediaType(image/png) = %q, want .png", got)
	}
	if got := extForMediaType("application/does-not-exist"); got != ".bin" {
		t.Errorf("extForMediaType(unknown) = %q, want .bin", got)
	}
}

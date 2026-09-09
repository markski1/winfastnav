package ui

import "testing"

func TestParseBasicMarkdown(t *testing.T) {
	spans := parseBasicMarkdown("Use **bold**, *italic*, [a link](https://example.com), and https://go.dev.")
	if len(spans) != 9 {
		t.Fatalf("span count = %d, want 9", len(spans))
	}
	if !spans[1].bold || spans[1].text != "bold" {
		t.Fatalf("bold span = %#v", spans[1])
	}
	if !spans[3].italic || spans[3].text != "italic" {
		t.Fatalf("italic span = %#v", spans[3])
	}
	if spans[5].url != "https://example.com" || spans[5].text != "a link" {
		t.Fatalf("link span = %#v", spans[5])
	}
	if spans[7].url != "https://go.dev" {
		t.Fatalf("URL span = %#v", spans[7])
	}
	if spans[8].text != "." {
		t.Fatalf("trailing punctuation = %#v", spans[8])
	}
}

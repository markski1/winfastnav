package core

import (
	"strings"
	"testing"
	"winfastnav/internal/globals"
)

func TestBareMathIsOfferedAsFirstResult(t *testing.T) {
	results, message := HandleTextInput("2 + 2")
	if message != nil {
		t.Fatalf("unexpected message: %q", *message)
	}
	if len(results) == 0 || !results[0].Computed || results[0].Name != "4" {
		t.Fatalf("first result = %#v, want computed 4", results)
	}
}

func TestBareUnitConversionIsOfferedAsFirstResult(t *testing.T) {
	results, message := HandleTextInput("20in")
	if message != nil {
		t.Fatalf("unexpected message: %q", *message)
	}
	if len(results) == 0 || !results[0].Computed || !strings.Contains(results[0].Name, "cm") {
		t.Fatalf("first result = %#v, want a computed conversion", results)
	}
}

func TestSystemCommandAppearsAsResult(t *testing.T) {
	results, message := HandleTextInput("restart")
	if message != nil {
		t.Fatalf("unexpected message: %q", *message)
	}
	if len(results) == 0 || results[0].Command == nil || results[0].Command.Action != "restart" {
		t.Fatalf("first result = %#v, want restart command", results)
	}
}

func TestWebSearchFallbackAppearsWhenNoProgramMatches(t *testing.T) {
	results, message := HandleTextInput("query-that-will-not-match-anything")
	if message != nil {
		t.Fatalf("unexpected message: %q", *message)
	}
	if len(results) != 2 || results[0].Assistant == "" || results[1].WebSearch == "" {
		t.Fatalf("fallback result = %#v", results)
	}
}

func TestCombineResultsPrioritizesAppsThenCommandsThenDocuments(t *testing.T) {
	results := combineResults(
		[]globals.Resource{{Name: "App"}},
		[]globals.Resource{{Name: "Command", Command: &globals.SystemCommand{Action: "test"}}},
		[]globals.Resource{{Name: "Document"}},
	)
	if len(results) != 3 || results[0].Name != "App" || results[1].Name != "Command" || results[2].Name != "Document" {
		t.Fatalf("priority order = %#v", results)
	}
}

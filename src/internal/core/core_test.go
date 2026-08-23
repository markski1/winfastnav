package core

import (
	"strings"
	"testing"
	"winfastnav/internal/globals"
)

func TestBareMathIsOfferedAsFirstResult(t *testing.T) {
	previousMode := globals.CurrentMode
	globals.CurrentMode = globals.ModeSearchProgram
	t.Cleanup(func() { globals.CurrentMode = previousMode })

	results, message := HandleTextInput("2 + 2")
	if message != nil {
		t.Fatalf("unexpected message: %q", *message)
	}
	if len(results) == 0 || !results[0].Computed || results[0].Name != "4" {
		t.Fatalf("first result = %#v, want computed 4", results)
	}
}

func TestBareUnitConversionIsOfferedAsFirstResult(t *testing.T) {
	previousMode := globals.CurrentMode
	globals.CurrentMode = globals.ModeSearchProgram
	t.Cleanup(func() { globals.CurrentMode = previousMode })

	results, message := HandleTextInput("20in")
	if message != nil {
		t.Fatalf("unexpected message: %q", *message)
	}
	if len(results) == 0 || !results[0].Computed || !strings.Contains(results[0].Name, "cm") {
		t.Fatalf("first result = %#v, want a computed conversion", results)
	}
}

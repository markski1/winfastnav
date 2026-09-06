package systemactions

import "testing"

func TestFindSystemAction(t *testing.T) {
	results := Find("wifi")
	if len(results) == 0 || results[0].Command == nil || results[0].Command.Action != "ms-settings:network-wifi" {
		t.Fatalf("Wi-Fi results = %#v", results)
	}
}

func TestDisruptiveActionsRequireConfirmation(t *testing.T) {
	for _, action := range []string{ActionLock, ActionSleep, ActionRestart, ActionShutdown, ActionEmptyRecycle} {
		if !RequiresConfirmation(action) {
			t.Errorf("action %q does not require confirmation", action)
		}
	}
	if RequiresConfirmation("ms-settings:display") {
		t.Fatal("opening Settings should not require confirmation")
	}
}

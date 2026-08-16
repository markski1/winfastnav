package presentation

import "testing"

func TestControllerUpdatesStateAndInvalidates(t *testing.T) {
	controller := NewController(10)
	t.Cleanup(controller.Close)

	invalidated := make(chan struct{}, 1)
	controller.SetInvalidator(func() {
		invalidated <- struct{}{}
	})

	state, ok := controller.Dispatch(Command{Kind: CommandSetResults, ResultCount: 3})
	if !ok {
		t.Fatal("controller stopped unexpectedly")
	}
	if state.ResultCount != 3 || state.Selected != -1 {
		t.Fatalf("unexpected result state: %+v", state)
	}

	state, ok = controller.Dispatch(Command{Kind: CommandSelectResult, Selected: 1})
	if !ok || state.Selected != 1 {
		t.Fatalf("unexpected selection state: %+v", state)
	}

	select {
	case <-invalidated:
	default:
		t.Fatal("state change did not request a redraw")
	}
}

func TestControllerClearsSelectionWhenQueryChanges(t *testing.T) {
	controller := NewController(10)
	t.Cleanup(controller.Close)

	_, _ = controller.Dispatch(Command{Kind: CommandSetResults, ResultCount: 2})
	_, _ = controller.Dispatch(Command{Kind: CommandSelectResult, Selected: 0})
	state, ok := controller.Dispatch(Command{Kind: CommandSetQuery, Query: "calc"})
	if !ok {
		t.Fatal("controller stopped unexpectedly")
	}
	if state.Query != "calc" || state.Selected != -1 {
		t.Fatalf("unexpected query state: %+v", state)
	}
}

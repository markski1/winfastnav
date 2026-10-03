package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"gioui.org/io/input"
	"gioui.org/io/key"
	"gioui.org/layout"
	appsettings "winfastnav/internal/settings"
)

func TestQuickAnswerControlsAndPreservation(t *testing.T) {
	type request struct {
		prompt string
		reply  chan string
	}
	requests := make(chan request, 2)
	l := pagesTestLauncher()
	l.state = uiState{page: pageLauncher, visible: true, query: "a question"}
	l.pageList.Axis = layout.Vertical
	l.answerFetch = func(prompt string) string {
		reply := make(chan string)
		requests <- request{prompt, reply}
		return <-reply
	}
	l.askAssistant("a question")
	first := <-requests
	l.prepareHide()
	l.prepareShow()
	if state := l.snapshot(); !state.answer || !state.loading || state.answerPrompt != first.prompt || state.query != first.prompt {
		t.Fatalf("reopening lost the pending answer: %+v", state)
	}
	first.reply <- "An **answer**. See [source](https://example.com)."
	waitForAnswer(t, l)
	l.prepareHide()
	l.prepareShow()
	var router input.Router
	gtx := settingsTestContext(1)
	gtx.Source = router.Source()
	l.answerCopy.Click()
	l.quickAnswer(gtx, l.snapshot().message)
	_, value, copied := router.WriteClipboard()
	if !copied || string(value) != "An answer. See source (https://example.com)." || !l.snapshot().answerCopied {
		t.Fatalf("answer clipboard = %q, copied=%v", value, copied)
	}
	l.answerRetry.Click()
	l.quickAnswer(gtx, l.snapshot().message)
	second := <-requests
	if second.prompt != first.prompt || !l.snapshot().loading || l.snapshot().answerCopied {
		t.Fatal("retry did not reuse the question and reset feedback")
	}
	l.copyAnswer(gtx)
	if _, _, copied := router.WriteClipboard(); copied {
		t.Fatal("copied a loading message")
	}
	second.reply <- "A fresh answer."
	waitForAnswer(t, l)
	if l.snapshot().message != "A fresh answer." {
		t.Fatal("retry did not replace the answer")
	}
	l.query("> echo hello")
	if state := l.snapshot(); state.answer || state.answerPrompt != "" {
		t.Fatal("new search kept the previous answer")
	}
}

func waitForAnswer(t *testing.T, l *launcher) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for l.snapshot().loading {
		if time.Now().After(deadline) {
			t.Fatal("answer did not complete")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestAnswerPlainText(t *testing.T) {
	got := answerPlainText("**Bold** and *italic*\r\nhttps://example.com")
	if got != "Bold and italic\nhttps://example.com" {
		t.Fatalf("plain answer = %q", got)
	}
}

func TestBackNavigationRemembersPane(t *testing.T) {
	l := settingsTestLauncher()
	l.helpPane, l.settingsPane = helpTools, settingsAppearance
	for _, page := range []page{pageSettings, pageHelp, pageAbout} {
		l.state.page = page
		l.key(settingsTestContext(1), key.Event{Name: key.NameEscape})
		if l.snapshot().page != pageMenu {
			t.Fatalf("Esc from %d did not return to Menu", page)
		}
	}
	l.help.Click()
	l.menuPage(settingsTestContext(1))
	if l.snapshot().page != pageHelp || l.helpPane != helpTools {
		t.Fatal("Help forgot its pane")
	}
	l.backPage()
	l.settingsButton.Click()
	l.menuPage(settingsTestContext(1))
	if l.snapshot().page != pageSettings || l.settingsPane != settingsAppearance {
		t.Fatal("Settings forgot its pane")
	}
	l.backPage()
	l.backPage()
	if l.snapshot().page != pageLauncher {
		t.Fatal("Menu did not return to search")
	}
}

func TestStartupToggleAndSaveFailure(t *testing.T) {
	l := settingsTestLauncher()
	var saved []bool
	l.startupSet = func(enabled bool) error { saved = append(saved, enabled); return nil }
	l.setStartup(true)
	l.setStartup(false)
	if len(saved) != 2 || !saved[0] || saved[1] || l.startupEnabled || l.startupSwitch.Value {
		t.Fatal("startup toggle did not save both states")
	}
	l.startupSet = func(bool) error { return errors.New("access denied") }
	l.startupSwitch.Value = true
	l.setStartup(true)
	if l.startupEnabled || l.startupSwitch.Value || !strings.Contains(l.settingsStatus, "Could not change") {
		t.Fatal("failed startup save was displayed as enabled")
	}
}

func TestStartupSwitchKeyboard(t *testing.T) {
	l := settingsTestLauncher()
	l.state.page = pageSettings
	var saved []bool
	l.startupSet = func(enabled bool) error { saved = append(saved, enabled); return nil }
	var router input.Router
	gtx := settingsTestContext(1)
	gtx.Source = router.Source()
	frame := func() {
		gtx.Ops.Reset()
		l.update(gtx)
		l.settingsPage(gtx)
		router.Frame(gtx.Ops)
	}
	frame()
	gtx.Execute(key.FocusCmd{Tag: &l.startupSwitch})
	frame()
	for range 2 {
		router.Queue(key.Event{Name: key.NameReturn, State: key.Press}, key.Event{Name: key.NameReturn, State: key.Release})
		frame()
		frame()
	}
	if len(saved) != 2 || !saved[0] || saved[1] {
		t.Fatalf("switch saves = %v", saved)
	}
}

func TestTextSizeAndPendingFolderFeedback(t *testing.T) {
	t.Setenv("APPDATA", t.TempDir())
	l := settingsTestLauncher()
	l.setTextSize(2)
	value, err := appsettings.GetSetting("textsize")
	if err != nil || value != "larger" || textSizeIndexForSetting(value) != 2 || textSizeScale(l.textSizeIndex) != 1.3 {
		t.Fatalf("text size did not persist: %q, %v", value, err)
	}
	l.indexRoots.SetText(`C:\Documents; D:\New`)
	if !strings.Contains(l.settingsSaveStatus(), "Text size saved.") || !strings.Contains(l.settingsSaveStatus(), "Folder changes pending.") {
		t.Fatalf("saved and pending feedback = %q", l.settingsSaveStatus())
	}
	l.indexRoots.SetText(`c:\documents; d:\projects`)
	if l.indexSettingsPending() {
		t.Fatal("equivalent folder paths displayed as pending")
	}
	t.Setenv("APPDATA", "")
	l.setTextSize(1)
	if l.textSizeIndex != 2 || !strings.Contains(l.settingsStatus, "Could not save text size") {
		t.Fatal("failed text save changed the preference")
	}
	l.indexRoots.SetText(`D:\Changed`)
	l.state.page = pageSettings
	l.backPage()
	if l.snapshot().page != pageSettings || !l.indexSettingsPending() || !strings.Contains(l.settingsStatus, "Could not save index") {
		t.Fatal("failed folder save discarded pending changes")
	}
}

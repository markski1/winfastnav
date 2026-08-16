//go:build windows

package hotkey

import (
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	modifierAlt = 0x0001
	virtualKeyO = 0x4f
	hotkeyID    = 1
	wmHotkey    = 0x0312
	wmQuit      = 0x0012
)

var (
	user32                 = windows.NewLazySystemDLL("user32.dll")
	kernel32               = windows.NewLazySystemDLL("kernel32.dll")
	procRegisterHotKey     = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey   = user32.NewProc("UnregisterHotKey")
	procGetMessage         = user32.NewProc("GetMessageW")
	procPostThreadMessage  = user32.NewProc("PostThreadMessageW")
	procGetCurrentThreadID = kernel32.NewProc("GetCurrentThreadId")
)

type point struct {
	X int32
	Y int32
}

type message struct {
	Window  windows.Handle
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Point   point
}

type Listener struct {
	mu       sync.Mutex
	threadID uint32
}

func Start(onHotkey func()) (*Listener, error) {
	listener := &Listener{}
	ready := make(chan error, 1)
	go listener.run(onHotkey, ready)
	if err := <-ready; err != nil {
		return nil, err
	}
	return listener, nil
}

func (l *Listener) Stop() {
	l.mu.Lock()
	threadID := l.threadID
	l.mu.Unlock()
	if threadID != 0 {
		procPostThreadMessage.Call(uintptr(threadID), wmQuit, 0, 0)
	}
}

func (l *Listener) run(onHotkey func(), ready chan<- error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	threadID, _, _ := procGetCurrentThreadID.Call()
	l.mu.Lock()
	l.threadID = uint32(threadID)
	l.mu.Unlock()

	registered, _, err := procRegisterHotKey.Call(0, hotkeyID, modifierAlt, virtualKeyO)
	if registered == 0 {
		ready <- fmt.Errorf("register Alt+O hotkey: %w", err)
		return
	}
	defer procUnregisterHotKey.Call(0, hotkeyID)
	ready <- nil

	for {
		var message message
		result, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if result == ^uintptr(0) || result == 0 {
			return
		}
		if message.Message == wmHotkey {
			onHotkey()
		}
	}
}

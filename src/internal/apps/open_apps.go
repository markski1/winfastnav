package apps

import (
	"fmt"
	"golang.org/x/sys/windows"
	"sort"
	"syscall"
	"unsafe"
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetWindowTextLength = user32.NewProc("GetWindowTextLengthW")
	procGetWindowText       = user32.NewProc("GetWindowTextW")
	procIsWindowVisible     = user32.NewProc("IsWindowVisible")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	lastOpenWindows         map[int]HWND
)

type HWND windows.Handle

// used for sort
type windowEntry struct {
	handle HWND
	title  string
}

func GetOpenWindows() string {
	var windowsMap = make(map[HWND]string)

	callback := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		h := HWND(hwnd)
		if isWindowVisible(h) {
			title := getWindowText(h)
			if len(title) > 0 {
				windowsMap[h] = title
			}
		}
		return 1
	})

	_, _, _ = procEnumWindows.Call(callback, 0)

	var retval string = ""
	lastOpenWindows = map[int]HWND{}
	count := 1

	// move to a windowEntry struct for sorting
	entries := make([]windowEntry, 0, len(windowsMap))
	for hwnd, title := range windowsMap {
		entries = append(entries, windowEntry{hwnd, title})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].title < entries[j].title
	})

	for _, entry := range entries {
		lastOpenWindows[count] = entry.handle
		// for unicode safety we use runes and not chars
		runes := []rune(entry.title)
		var showTitle string
		if len(runes) > 64 {
			showTitle = string(runes[:60])
		} else {
			showTitle = entry.title
		}
		retval += fmt.Sprintf("[ %v ] %v\n", count, showTitle)
		count++
	}

	return retval
}

func FocusWindow(windowNumber int) {
	if windowNumber > 0 && windowNumber <= len(lastOpenWindows) {
		h := lastOpenWindows[windowNumber]
		setForegroundWindow(h)
	}
}

func getWindowText(hwnd HWND) string {
	length, _, _ := procGetWindowTextLength.Call(uintptr(hwnd))
	if length == 0 {
		return ""
	}
	buf := make([]uint16, length+1)
	_, _, _ = procGetWindowText.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), length+1)
	return syscall.UTF16ToString(buf)
}

func isWindowVisible(hwnd HWND) bool {
	ret, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return ret != 0
}

func setForegroundWindow(hwnd HWND) bool {
	ret, _, _ := procSetForegroundWindow.Call(uintptr(hwnd))
	return ret != 0
}

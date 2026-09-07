package systemactions

import (
	"errors"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"

	g "winfastnav/internal/globals"
	"winfastnav/internal/recent"
)

const (
	ActionLock         = "lock"
	ActionSleep        = "sleep"
	ActionRestart      = "restart"
	ActionShutdown     = "shutdown"
	ActionEmptyRecycle = "empty-recycle-bin"
)

var (
	user32               = windows.NewLazySystemDLL("user32.dll")
	powrprof             = windows.NewLazySystemDLL("powrprof.dll")
	shell32              = windows.NewLazySystemDLL("shell32.dll")
	lockWorkStation      = user32.NewProc("LockWorkStation")
	setSuspendState      = powrprof.NewProc("SetSuspendState")
	shEmptyRecycleBin    = shell32.NewProc("SHEmptyRecycleBinW")
	availableSystemItems = prepare([]g.Resource{
		command("Lock computer", ActionLock, "Lock this Windows session"),
		command("Sleep computer", ActionSleep, "Put the computer to sleep"),
		command("Restart computer", ActionRestart, "Restart Windows"),
		command("Shut down computer", ActionShutdown, "Turn off the computer"),
		command("Empty Recycle Bin", ActionEmptyRecycle, "Permanently delete recycled files"),
		command("Windows Settings", "ms-settings:", "Open Windows Settings"),
		command("Bluetooth settings", "ms-settings:bluetooth", "Open Bluetooth settings"),
		command("Wi-Fi settings", "ms-settings:network-wifi", "Open Wi-Fi settings"),
		command("Display settings", "ms-settings:display", "Open display settings"),
		command("Sound settings", "ms-settings:sound", "Open sound settings"),
		command("Notification settings", "ms-settings:notifications", "Open notification settings"),
	})
)

func Find(query string) []g.Resource {
	query = strings.ToLower(strings.Join(strings.Fields(query), " "))
	if query == "" {
		return nil
	}
	matches := make([]g.Resource, 0, 6)
	for _, item := range availableSystemItems {
		if strings.Contains(item.SearchName, query) || strings.Contains(item.SearchPath, query) {
			matches = append(matches, item)
		}
	}
	matches = recent.MatchAndRankLimit(matches, query, 6)
	if len(matches) > 6 {
		matches = matches[:6]
	}
	return matches
}

func RequiresConfirmation(action string) bool {
	switch action {
	case ActionLock, ActionSleep, ActionRestart, ActionShutdown, ActionEmptyRecycle:
		return true
	default:
		return false
	}
}

func Execute(action string) error {
	switch action {
	case ActionLock:
		result, _, callErr := lockWorkStation.Call()
		return callResult(result, callErr)
	case ActionSleep:
		result, _, callErr := setSuspendState.Call(0, 0, 0)
		return callResult(result, callErr)
	case ActionRestart:
		return start("shutdown.exe", "/r", "/t", "0")
	case ActionShutdown:
		return start("shutdown.exe", "/s", "/t", "0")
	case ActionEmptyRecycle:
		const noUI = 0x00000001 | 0x00000002 | 0x00000004
		result, _, callErr := shEmptyRecycleBin.Call(0, 0, noUI)
		if result != 0 {
			return callError(callErr)
		}
		return nil
	default:
		if strings.HasPrefix(action, "ms-settings:") {
			return start("explorer.exe", action)
		}
		return errors.New("unknown system action")
	}
}

func prepare(items []g.Resource) []g.Resource {
	for index := range items {
		items[index].Filepath = "command:" + items[index].Command.Action
		items[index].SearchName = strings.ToLower(items[index].Name)
		items[index].SearchPath = strings.ToLower(items[index].Filepath + " " + items[index].Command.Detail)
	}
	return items
}

func command(name, action, detail string) g.Resource {
	return g.Resource{Name: name, Command: &g.SystemCommand{Action: action, Detail: detail}}
}

func start(name string, arguments ...string) error {
	command := exec.Command(name, arguments...)
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return command.Start()
}

func callResult(result uintptr, callErr error) error {
	if result != 0 {
		return nil
	}
	return callError(callErr)
}

func callError(callErr error) error {
	if callErr == nil || errors.Is(callErr, syscall.Errno(0)) {
		return errors.New("Windows rejected the system action")
	}
	return callErr
}

//go:build windows

package instance

import (
	"errors"

	"golang.org/x/sys/windows"
)

// Guard keeps one launcher process active for the current Windows session.
type Guard struct {
	handle windows.Handle
}

func Acquire(name string) (*Guard, bool, error) {
	namePtr, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, false, err
	}

	handle, err := windows.CreateMutex(nil, false, namePtr)
	if handle == 0 {
		if err == nil {
			err = errors.New("CreateMutex returned a null handle")
		}
		return nil, false, err
	}
	if err == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(handle)
		return nil, false, nil
	}
	if err != nil {
		_ = windows.CloseHandle(handle)
		return nil, false, err
	}

	return &Guard{handle: handle}, true, nil
}

func (g *Guard) Release() {
	if g == nil || g.handle == 0 {
		return
	}
	_ = windows.CloseHandle(g.handle)
	g.handle = 0
}

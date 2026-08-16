//go:build windows

package windowcontrol

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	swHide    = 0
	swRestore = 9
)

var (
	user32                  = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows         = user32.NewProc("EnumWindows")
	procGetWindowTextLength = user32.NewProc("GetWindowTextLengthW")
	procGetWindowText       = user32.NewProc("GetWindowTextW")
	procGetWindowProcessID  = user32.NewProc("GetWindowThreadProcessId")
	procIsWindow            = user32.NewProc("IsWindow")
	procShowWindow          = user32.NewProc("ShowWindow")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
)

type Controller struct {
	mu     sync.Mutex
	title  string
	handle windows.Handle
}

func New(title string) *Controller { return &Controller{title: title} }

func (c *Controller) BindView(view any) {
	value := reflect.ValueOf(view)
	if value.Kind() != reflect.Struct {
		return
	}
	handle := value.FieldByName("HWND")
	if !handle.IsValid() || handle.Kind() != reflect.Uintptr {
		return
	}
	c.mu.Lock()
	c.handle = windows.Handle(handle.Uint())
	c.mu.Unlock()
}

func (c *Controller) Bind() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.isValid(c.handle) {
		return nil
	}
	pID := uint32(os.Getpid())
	var found windows.Handle
	callback := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		handle := windows.Handle(hwnd)
		if c.windowProcessID(handle) != pID || c.windowTitle(handle) != c.title {
			return 1
		}
		found = handle
		return 0
	})
	procEnumWindows.Call(callback, 0)
	if found == 0 {
		return fmt.Errorf("could not find Gio window %q", c.title)
	}
	c.handle = found
	return nil
}

func (c *Controller) Hide() error {
	if err := c.Bind(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	procShowWindow.Call(uintptr(c.handle), swHide)
	return nil
}

func (c *Controller) ShowAndFocus() error {
	if err := c.Bind(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	procShowWindow.Call(uintptr(c.handle), swRestore)
	procSetForegroundWindow.Call(uintptr(c.handle))
	return nil
}

func (c *Controller) isValid(handle windows.Handle) bool {
	if handle == 0 {
		return false
	}
	result, _, _ := procIsWindow.Call(uintptr(handle))
	return result != 0
}

func (c *Controller) windowProcessID(handle windows.Handle) uint32 {
	var pID uint32
	procGetWindowProcessID.Call(uintptr(handle), uintptr(unsafe.Pointer(&pID)))
	return pID
}

func (c *Controller) windowTitle(handle windows.Handle) string {
	length, _, _ := procGetWindowTextLength.Call(uintptr(handle))
	if length == 0 {
		return ""
	}
	buffer := make([]uint16, length+1)
	procGetWindowText.Call(uintptr(handle), uintptr(unsafe.Pointer(&buffer[0])), length+1)
	return syscall.UTF16ToString(buffer)
}

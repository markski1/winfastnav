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
	swHide                  = 0
	swRestore               = 9
	monitorDefaultToNearest = 2
	swpNoZOrder             = 0x0004
	swpNoActivate           = 0x0010
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
	procGetForegroundWindow = user32.NewProc("GetForegroundWindow")
	procGetWindowRect       = user32.NewProc("GetWindowRect")
	procMonitorFromWindow   = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfo      = user32.NewProc("GetMonitorInfoW")
	procSetWindowPos        = user32.NewProc("SetWindowPos")
)

type rect struct {
	left, top, right, bottom int32
}

type monitorInfo struct {
	size    uint32
	monitor rect
	work    rect
	flags   uint32
}

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
		if c.windowProcessID(handle) != pID || windowTitle(handle) != c.title {
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
	_ = c.CenterOnForegroundMonitor()
	c.mu.Lock()
	defer c.mu.Unlock()
	procShowWindow.Call(uintptr(c.handle), swRestore)
	procSetForegroundWindow.Call(uintptr(c.handle))
	return nil
}

func ShowExistingAndFocus(title string) (bool, error) {
	if title == "" {
		return false, fmt.Errorf("window title is empty")
	}

	var found windows.Handle
	callback := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		handle := windows.Handle(hwnd)
		if windowTitle(handle) != title {
			return 1
		}
		found = handle
		return 0
	})
	procEnumWindows.Call(callback, 0)
	if found == 0 {
		return false, nil
	}

	procShowWindow.Call(uintptr(found), swRestore)
	procSetForegroundWindow.Call(uintptr(found))
	return true, nil
}

// CenterOnForegroundMonitor moves the launcher to the work area of the monitor
// that was active before the launcher was shown.
func (c *Controller) CenterOnForegroundMonitor() error {
	if err := c.Bind(); err != nil {
		return err
	}

	c.mu.Lock()
	handle := c.handle
	c.mu.Unlock()

	var windowRect rect
	if result, _, _ := procGetWindowRect.Call(uintptr(handle), uintptr(unsafe.Pointer(&windowRect))); result == 0 {
		return fmt.Errorf("get launcher window rectangle")
	}
	width := windowRect.right - windowRect.left
	height := windowRect.bottom - windowRect.top
	if width <= 0 || height <= 0 {
		return nil
	}

	foreground, _, _ := procGetForegroundWindow.Call()
	if foreground == 0 {
		foreground = uintptr(handle)
	}
	monitor, _, _ := procMonitorFromWindow.Call(foreground, monitorDefaultToNearest)
	if monitor == 0 {
		return nil
	}

	info := monitorInfo{size: uint32(unsafe.Sizeof(monitorInfo{}))}
	if result, _, _ := procGetMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info))); result == 0 {
		return fmt.Errorf("get foreground monitor information")
	}

	x := info.work.left + (info.work.right-info.work.left-width)/2
	y := info.work.top + (info.work.bottom-info.work.top-height)/2
	procSetWindowPos.Call(uintptr(handle), 0, uintptr(x), uintptr(y), uintptr(width), uintptr(height), swpNoZOrder|swpNoActivate)
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

func windowTitle(handle windows.Handle) string {
	length, _, _ := procGetWindowTextLength.Call(uintptr(handle))
	if length == 0 {
		return ""
	}
	buffer := make([]uint16, length+1)
	procGetWindowText.Call(uintptr(handle), uintptr(unsafe.Pointer(&buffer[0])), length+1)
	return syscall.UTF16ToString(buffer)
}

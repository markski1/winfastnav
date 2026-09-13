package utils

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const maxHTTPResponseSize = 1 << 20

var httpClient = &http.Client{Timeout: 15 * time.Second}

var (
	shell32                    = windows.NewLazySystemDLL("shell32.dll")
	ole32                      = windows.NewLazySystemDLL("ole32.dll")
	shParseDisplayName         = shell32.NewProc("SHParseDisplayName")
	shOpenFolderAndSelectItems = shell32.NewProc("SHOpenFolderAndSelectItems")
	coInitializeEx             = ole32.NewProc("CoInitializeEx")
	coUninitialize             = ole32.NewProc("CoUninitialize")
	coTaskMemFree              = ole32.NewProc("CoTaskMemFree")
)

func HttpGet(url string) (string, error) {
	resp, err := httpClient.Get(url)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("request failed: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPResponseSize+1))
	if err != nil {
		return "", err
	}
	if len(body) > maxHTTPResponseSize {
		return "", fmt.Errorf("response exceeds %d bytes", maxHTTPResponseSize)
	}
	return string(body), nil
}

func WrapTextByWords(s string, maxLen int) string {
	if maxLen <= 0 {
		return s
	}

	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = wrapLine(line, maxLen)
	}
	return strings.Join(lines, "\n")
}

func wrapLine(line string, maxLen int) string {
	words := strings.Fields(line)
	if len(words) == 0 {
		return line
	}

	var b strings.Builder
	var lineLen int
	for _, w := range words {
		wLen := len([]rune(w))
		if lineLen == 0 {
			b.WriteString(w)
			lineLen = wLen
		} else if lineLen+1+wLen <= maxLen {
			b.WriteByte(' ')
			b.WriteString(w)
			lineLen += 1 + wLen
		} else {
			b.WriteRune('\n')
			b.WriteString(w)
			lineLen = wLen
		}
	}
	return b.String()
}

func ContainsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func AddToStartup() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.SET_VALUE)
	if err != nil {
		return err
	}

	err = key.SetStringValue("WinFastNav", exePath)
	_ = key.Close()
	return err
}

func IsInStartup() bool {
	exePath, err := os.Executable()
	if err != nil {
		return false
	}
	key, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`,
		registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()
	value, _, err := key.GetStringValue("WinFastNav")
	return err == nil && strings.EqualFold(value, exePath)
}

func RevealInFolder(path string) error {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	initialized, _, _ := coInitializeEx.Call(0, 2)
	if initialized == 0 || initialized == 1 {
		defer coUninitialize.Call()
	} else if uint32(initialized) != 0x80010106 {
		return fmt.Errorf("initialize COM: HRESULT 0x%08X", uint32(initialized))
	}

	var item uintptr
	result, _, _ := shParseDisplayName.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(&item)),
		0,
		0,
	)
	if int32(result) < 0 {
		return fmt.Errorf("resolve Explorer item: HRESULT 0x%08X", uint32(result))
	}
	defer coTaskMemFree.Call(item)

	result, _, _ = shOpenFolderAndSelectItems.Call(item, 0, 0, 0)
	if int32(result) < 0 {
		return fmt.Errorf("open Explorer selection: HRESULT 0x%08X", uint32(result))
	}
	return nil
}

func OpenURI(uri string) error {
	log.Printf("Opening URI: %s", uri)
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", uri)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow: true,
	}
	return cmd.Start()
}

func RunShellCommand(command string) error {
	cmd := exec.Command("cmd.exe", "/C", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Start()
}

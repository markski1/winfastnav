//go:build windows

package icons

import (
	"image"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	shgfiIcon          = 0x000000100
	shgfiShellIconSize = 0x000000004
	dibRGBColors       = 0
)

var (
	shell32           = windows.NewLazySystemDLL("shell32.dll")
	user32            = windows.NewLazySystemDLL("user32.dll")
	gdi32             = windows.NewLazySystemDLL("gdi32.dll")
	procSHGetFileInfo = shell32.NewProc("SHGetFileInfoW")
	procDestroyIcon   = user32.NewProc("DestroyIcon")
	procGetIconInfo   = user32.NewProc("GetIconInfo")
	procGetDC         = user32.NewProc("GetDC")
	procReleaseDC     = user32.NewProc("ReleaseDC")
	procDeleteObject  = gdi32.NewProc("DeleteObject")
	procGetObject     = gdi32.NewProc("GetObjectW")
	procGetDIBits     = gdi32.NewProc("GetDIBits")
)

type shellFileInfo struct {
	Icon        windows.Handle
	IconIndex   int32
	Attributes  uint32
	DisplayName [260]uint16
	TypeName    [80]uint16
}

type iconInfo struct {
	Icon     int32
	HotspotX uint32
	HotspotY uint32
	Mask     windows.Handle
	Color    windows.Handle
}

type bitmap struct {
	Type       int32
	Width      int32
	Height     int32
	WidthBytes int32
	Planes     uint16
	BitsPixel  uint16
	Bits       unsafe.Pointer
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type bitmapInfo struct {
	Header bitmapInfoHeader
	Colors [1]uint32
}

func load(path string) image.Image {
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil
	}

	var fileInfo shellFileInfo
	result, _, _ := procSHGetFileInfo.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		uintptr(unsafe.Pointer(&fileInfo)),
		unsafe.Sizeof(fileInfo),
		shgfiIcon|shgfiShellIconSize,
	)
	if result == 0 || fileInfo.Icon == 0 {
		return nil
	}
	defer procDestroyIcon.Call(uintptr(fileInfo.Icon))

	var details iconInfo
	gotInfo, _, _ := procGetIconInfo.Call(uintptr(fileInfo.Icon), uintptr(unsafe.Pointer(&details)))
	if gotInfo == 0 || details.Color == 0 {
		return nil
	}
	defer procDeleteObject.Call(uintptr(details.Color))
	if details.Mask != 0 {
		defer procDeleteObject.Call(uintptr(details.Mask))
	}

	var source bitmap
	if got, _, _ := procGetObject.Call(uintptr(details.Color), unsafe.Sizeof(source), uintptr(unsafe.Pointer(&source))); got == 0 || source.Width <= 0 || source.Height <= 0 {
		return nil
	}

	width, height := int(source.Width), int(source.Height)
	raw := make([]byte, width*height*4)
	info := bitmapInfo{Header: bitmapInfoHeader{
		Size:     uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		Width:    source.Width,
		Height:   -source.Height,
		Planes:   1,
		BitCount: 32,
	}}
	dc, _, _ := procGetDC.Call(0)
	if dc == 0 {
		return nil
	}
	defer procReleaseDC.Call(0, dc)
	if lines, _, _ := procGetDIBits.Call(dc, uintptr(details.Color), 0, uintptr(height), uintptr(unsafe.Pointer(&raw[0])), uintptr(unsafe.Pointer(&info)), dibRGBColors); lines == 0 {
		return nil
	}

	hasAlpha := false
	for offset := 3; offset < len(raw); offset += 4 {
		if raw[offset] != 0 {
			hasAlpha = true
			break
		}
	}
	converted := image.NewNRGBA(image.Rect(0, 0, width, height))
	for index := 0; index < width*height; index++ {
		sourceOffset := index * 4
		destinationOffset := index * 4
		converted.Pix[destinationOffset] = raw[sourceOffset+2]
		converted.Pix[destinationOffset+1] = raw[sourceOffset+1]
		converted.Pix[destinationOffset+2] = raw[sourceOffset]
		if hasAlpha {
			converted.Pix[destinationOffset+3] = raw[sourceOffset+3]
		} else {
			converted.Pix[destinationOffset+3] = 0xff
		}
	}
	return converted
}

var _ = syscall.Errno(0)

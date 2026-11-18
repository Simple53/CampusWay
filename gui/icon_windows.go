//go:build windows

package main

import (
	_ "embed"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

//go:embed campusway.ico
var appIconBytes []byte

const (
	wmSetIcon      = 0x0080
	iconSmall      = 0
	iconBig        = 1
	imageIcon      = 1
	lrLoadFromFile = 0x00000010
)

var (
	procFindWindowW  = user32.NewProc("FindWindowW")
	procSendMessageW = user32.NewProc("SendMessageW")
	procLoadImageW   = user32.NewProc("LoadImageW")

	cachedIcoPath string
	iconMu        sync.Mutex
	cachedIcon16  windows.Handle
	cachedIcon32  windows.Handle
)

// ensureIconFiles 将嵌入的 ico 写入临时目录并通过 LoadImageW 加载 16x16 与 32x32 图标句柄
func ensureIconFiles() (windows.Handle, windows.Handle) {
	iconMu.Lock()
	defer iconMu.Unlock()
	if cachedIcon16 != 0 && cachedIcon32 != 0 {
		return cachedIcon16, cachedIcon32
	}
	if len(appIconBytes) == 0 {
		return 0, 0
	}
	dir := os.TempDir()
	icoPath := filepath.Join(dir, "campusway_app_icon.ico")
	_ = os.WriteFile(icoPath, appIconBytes, 0644)
	cachedIcoPath = icoPath

	pathPtr, err := windows.UTF16PtrFromString(icoPath)
	if err != nil {
		return 0, 0
	}

	h16, _, _ := procLoadImageW.Call(
		0,
		uintptr(unsafe.Pointer(pathPtr)),
		imageIcon,
		16, 16,
		lrLoadFromFile,
	)
	h32, _, _ := procLoadImageW.Call(
		0,
		uintptr(unsafe.Pointer(pathPtr)),
		imageIcon,
		32, 32,
		lrLoadFromFile,
	)

	cachedIcon16 = windows.Handle(h16)
	cachedIcon32 = windows.Handle(h32)
	return cachedIcon16, cachedIcon32
}

// applyWindowIcons 将小图标与大图标显式注入当前 RouteDesk 顶层窗口，解决放大/最小化与左上角图标丢失问题
func applyWindowIcons() {
	h16, h32 := ensureIconFiles()
	if h16 == 0 && h32 == 0 {
		return
	}
	go func() {
		titlePtr, _ := windows.UTF16PtrFromString("RouteDesk")
		for i := 0; i < 30; i++ {
			hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(titlePtr)))
			if hwnd != 0 {
				if h16 != 0 {
					procSendMessageW.Call(hwnd, wmSetIcon, iconSmall, uintptr(h16))
				}
				if h32 != 0 {
					procSendMessageW.Call(hwnd, wmSetIcon, iconBig, uintptr(h32))
				}
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
}

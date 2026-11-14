//go:build windows

package main

import (
	"context"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows"
)

const (
	wmUser            = 0x0400
	wmTrayIcon        = wmUser + 1
	nimAdd            = 0x00000000
	nimModify         = 0x00000001
	nimDelete         = 0x00000002
	nifMessage        = 0x00000001
	nifIcon           = 0x00000002
	nifTip            = 0x00000004
	wmRButtonUp       = 0x0205
	wmLButtonUp       = 0x0202
	tpmRightButton    = 0x0002
	mfString          = 0x00000000
	mfSeparator       = 0x00000800
	cmdShow           = 1001
	cmdDisconnect     = 1002
	cmdQuit           = 1003
	idiApplication    = 32512
	wsOverlappedWindow = 0x00CF0000
)

type notifyIconData struct {
	cbSize           uint32
	hWnd             windows.HWND
	uID              uint32
	uFlags           uint32
	uCallbackMessage uint32
	hIcon            windows.Handle
	szTip            [128]uint16
}

type point struct {
	x, y int32
}

type msg struct {
	hwnd    windows.HWND
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

type wndClassEx struct {
	cbSize        uint32
	style         uint32
	lpfnWndProc   uintptr
	cbClsExtra    int32
	cbWndExtra    int32
	hInstance     windows.Handle
	hIcon         windows.Handle
	hCursor       windows.Handle
	hbrBackground windows.Handle
	lpszMenuName  *uint16
	lpszClassName *uint16
	hIconSm       windows.Handle
}

var (
	kernel32             = windows.NewLazySystemDLL("kernel32.dll")
	user32               = windows.NewLazySystemDLL("user32.dll")
	shell32              = windows.NewLazySystemDLL("shell32.dll")
	procGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")
	procRegisterClassExW = user32.NewProc("RegisterClassExW")
	procCreateWindowExW  = user32.NewProc("CreateWindowExW")
	procDefWindowProcW   = user32.NewProc("DefWindowProcW")
	procDestroyWindow    = user32.NewProc("DestroyWindow")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procGetMessageW      = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessageW = user32.NewProc("DispatchMessageW")
	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
	procCreatePopupMenu  = user32.NewProc("CreatePopupMenu")
	procAppendMenuW      = user32.NewProc("AppendMenuW")
	procTrackPopupMenu   = user32.NewProc("TrackPopupMenu")
	procDestroyMenu      = user32.NewProc("DestroyMenu")
	procGetCursorPos     = user32.NewProc("GetCursorPos")
	procSetForegroundWindow = user32.NewProc("SetForegroundWindow")
	procLoadIconW        = user32.NewProc("LoadIconW")
	procPostMessageW     = user32.NewProc("PostMessageW")
)

type systemTray struct {
	ctx      context.Context
	app      *AppServer
	nid      notifyIconData
	hwnd     windows.HWND
	onExit   func()
}

var globalTray *systemTray

func setupTray(ctx context.Context, app *AppServer, onExit func()) *systemTray {
	st := &systemTray{
		ctx:    ctx,
		app:    app,
		onExit: onExit,
	}
	globalTray = st
	go st.runMessageLoop()
	return st
}

func trayWndProc(hwnd windows.HWND, uMsg uint32, wParam uintptr, lParam uintptr) uintptr {
	if uMsg == wmTrayIcon && globalTray != nil {
		switch lParam {
		case wmLButtonUp:
			// Click tray icon to show window
			runtime.WindowUnminimise(globalTray.ctx)
			runtime.WindowShow(globalTray.ctx)
			return 0
		case wmRButtonUp:
			globalTray.showContextMenu()
			return 0
		}
	}
	r, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(uMsg), wParam, lParam)
	return r
}

func (t *systemTray) showContextMenu() {
	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	showStr, _ := windows.UTF16PtrFromString("显示主窗口 (Show Window)")
	procAppendMenuW.Call(hMenu, mfString, cmdShow, uintptr(unsafe.Pointer(showStr)))

	// Check if connected
	snap := t.app.GetSnapshot()
	if snap.State == "connected" || snap.State == "connecting" {
		dcStr, _ := windows.UTF16PtrFromString("断开连接 (Disconnect)")
		procAppendMenuW.Call(hMenu, mfString, cmdDisconnect, uintptr(unsafe.Pointer(dcStr)))
	}

	procAppendMenuW.Call(hMenu, mfSeparator, 0, 0)
	quitStr, _ := windows.UTF16PtrFromString("退出 CampusWay (Quit)")
	procAppendMenuW.Call(hMenu, mfString, cmdQuit, uintptr(unsafe.Pointer(quitStr)))

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetForegroundWindow.Call(uintptr(t.hwnd))

	cmd, _, _ := procTrackPopupMenu.Call(
		hMenu,
		uintptr(tpmRightButton|0x0100), // TPM_RETURNCMD
		uintptr(pt.x),
		uintptr(pt.y),
		0,
		uintptr(t.hwnd),
		0,
	)

	switch cmd {
	case cmdShow:
		runtime.WindowUnminimise(t.ctx)
		runtime.WindowShow(t.ctx)
	case cmdDisconnect:
		_ = t.app.Disconnect()
	case cmdQuit:
		if t.onExit != nil {
			t.onExit()
		}
	}
}

func (t *systemTray) runMessageLoop() {
	className, _ := windows.UTF16PtrFromString("CampusWayTrayClass")
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	hIcon, _, _ := procLoadIconW.Call(0, idiApplication)

	var wc wndClassEx
	wc.cbSize = uint32(unsafe.Sizeof(wc))
	wc.lpfnWndProc = syscall.NewCallback(trayWndProc)
	wc.hInstance = windows.Handle(hInstance)
	wc.lpszClassName = className
	wc.hIcon = windows.Handle(hIcon)

	atom, _, _ := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	if atom == 0 {
		return
	}

	hwnd, _, _ := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		0,
		0,
		0, 0, 0, 0,
		0, 0,
		uintptr(hInstance),
		0,
	)
	if hwnd == 0 {
		return
	}
	t.hwnd = windows.HWND(hwnd)

	// Add tray icon
	t.nid.cbSize = uint32(unsafe.Sizeof(t.nid))
	t.nid.hWnd = t.hwnd
	t.nid.uID = 1
	t.nid.uFlags = nifMessage | nifIcon | nifTip
	t.nid.uCallbackMessage = wmTrayIcon
	t.nid.hIcon = windows.Handle(hIcon)
	tip, _ := windows.UTF16FromString("CampusWay")
	copy(t.nid.szTip[:], tip)

	procShellNotifyIconW.Call(nimAdd, uintptr(unsafe.Pointer(&t.nid)))

	var m msg
	for {
		res, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(res) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}

	procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&t.nid)))
	procDestroyWindow.Call(uintptr(t.hwnd))
}

func (t *systemTray) destroy() {
	if t.hwnd != 0 {
		procShellNotifyIconW.Call(nimDelete, uintptr(unsafe.Pointer(&t.nid)))
		procPostMessageW.Call(uintptr(t.hwnd), 0x0012, 0, 0) // WM_QUIT
	}
}

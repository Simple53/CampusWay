//go:build windows && legacy

package main

import (
	"fmt"
	"os/exec"
	"syscall"
	"unsafe"
)

var (
	modUser32   = syscall.NewLazyDLL("user32.dll")
	modShell32  = syscall.NewLazyDLL("shell32.dll")
	modKernel32 = syscall.NewLazyDLL("kernel32.dll")

	procRegisterClassExW    = modUser32.NewProc("RegisterClassExW")
	procCreateWindowExW     = modUser32.NewProc("CreateWindowExW")
	procDefWindowProcW      = modUser32.NewProc("DefWindowProcW")
	procGetMessageW         = modUser32.NewProc("GetMessageW")
	procTranslateMessage    = modUser32.NewProc("TranslateMessage")
	procDispatchMessageW    = modUser32.NewProc("DispatchMessageW")
	procPostQuitMessage     = modUser32.NewProc("PostQuitMessage")
	procDestroyWindow       = modUser32.NewProc("DestroyWindow")
	procCreatePopupMenu     = modUser32.NewProc("CreatePopupMenu")
	procAppendMenuW         = modUser32.NewProc("AppendMenuW")
	procTrackPopupMenu      = modUser32.NewProc("TrackPopupMenu")
	procGetCursorPos        = modUser32.NewProc("GetCursorPos")
	procSetForegroundWindow = modUser32.NewProc("SetForegroundWindow")
	procDestroyMenu         = modUser32.NewProc("DestroyMenu")
	procLoadIconW           = modUser32.NewProc("LoadIconW")

	procShell_NotifyIconW = modShell32.NewProc("Shell_NotifyIconW")
	procGetModuleHandleW  = modKernel32.NewProc("GetModuleHandleW")
)

const (
	WM_USER          = 0x0400
	WM_TRAYICON      = WM_USER + 100
	WM_LBUTTONUP     = 0x0202
	WM_RBUTTONUP     = 0x0205
	WM_LBUTTONDBLCLK = 0x0203
	WM_COMMAND       = 0x0111
	WM_DESTROY       = 0x0002

	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004
	NIF_INFO    = 0x00000010

	MF_STRING       = 0x00000000
	MF_SEPARATOR    = 0x00000800
	TPM_BOTTOMALIGN = 0x00000020
	TPM_RIGHTALIGN  = 0x00000008

	IDI_APPLICATION = 32512

	CMD_SHOW_DASHBOARD = 1001
	CMD_CONNECT        = 1002
	CMD_DISCONNECT     = 1003
	CMD_EXIT           = 1004
)

type POINT struct {
	X int32
	Y int32
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type NOTIFYICONDATAW struct {
	CbSize            uint32
	HWnd              uintptr
	UID               uint32
	UFlags            uint32
	UCallbackMessage  uint32
	HIcon             uintptr
	SzTip             [128]uint16
	DwState           uint32
	DwStateMask       uint32
	SzInfo            [256]uint16
	UTimeoutOrVersion uint32
	SzInfoTitle       [64]uint16
	DwInfoFlags       uint32
	GuidItem          [16]byte
	HBalloonIcon      uintptr
}

type TrayManager struct {
	hwnd      uintptr
	nid       NOTIFYICONDATAW
	server    *AppServer
	serverURL string
}

var globalTray *TrayManager

func stringToUTF16Ptr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func copyToUTF16Array(dest []uint16, src string) {
	utf16, _ := syscall.UTF16FromString(src)
	copy(dest, utf16)
}

func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	if globalTray == nil {
		r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
		return r
	}

	switch msg {
	case WM_TRAYICON:
		switch lParam {
		case WM_LBUTTONUP, WM_LBUTTONDBLCLK:
			globalTray.OpenDashboard()
		case WM_RBUTTONUP:
			globalTray.ShowContextMenu()
		}
		return 0

	case WM_COMMAND:
		cmdId := uint32(wParam & 0xFFFF)
		switch cmdId {
		case CMD_SHOW_DASHBOARD:
			globalTray.OpenDashboard()
		case CMD_CONNECT:
			_ = globalTray.server.StartVPN(globalTray.server.CurrentConfig())
		case CMD_DISCONNECT:
			globalTray.server.StopVPN()
		case CMD_EXIT:
			globalTray.server.StopVPN()
			globalTray.RemoveIcon()
			procPostQuitMessage.Call(0)
		}
		return 0

	case WM_DESTROY:
		globalTray.RemoveIcon()
		procPostQuitMessage.Call(0)
		return 0
	}

	r, _, _ := procDefWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func InitTray(server *AppServer, serverURL string) (*TrayManager, error) {
	className := "NNUConnectTrayHelperClass"
	hInstance, _, _ := procGetModuleHandleW.Call(0)
	// Resource ID 1 is generated from gui/assets/nnu-connect.ico by rsrc.
	hIcon, _, _ := procLoadIconW.Call(hInstance, 1)
	if hIcon == 0 {
		hIcon, _, _ = procLoadIconW.Call(0, uintptr(IDI_APPLICATION))
	}

	wcx := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     hInstance,
		HIcon:         hIcon,
		LpszClassName: stringToUTF16Ptr(className),
	}

	atom, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(&wcx)))
	if atom == 0 {
		return nil, fmt.Errorf("RegisterClassExW failed: %v", err)
	}

	hwnd, _, err := procCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(stringToUTF16Ptr(className))),
		uintptr(unsafe.Pointer(stringToUTF16Ptr("NNU-Connect Tray Helper"))),
		0,
		0, 0, 0, 0,
		0, 0, hInstance, 0,
	)
	if hwnd == 0 {
		return nil, fmt.Errorf("CreateWindowExW failed: %v", err)
	}

	tm := &TrayManager{
		hwnd:      hwnd,
		server:    server,
		serverURL: serverURL,
	}
	globalTray = tm

	tm.nid.CbSize = uint32(unsafe.Sizeof(tm.nid))
	tm.nid.HWnd = hwnd
	tm.nid.UID = 1
	tm.nid.UFlags = NIF_MESSAGE | NIF_ICON | NIF_TIP
	tm.nid.UCallbackMessage = WM_TRAYICON
	tm.nid.HIcon = hIcon
	copyToUTF16Array(tm.nid.SzTip[:], "NNU-Connect (南京师范大学 VPN) - 未连接")

	procShell_NotifyIconW.Call(NIM_ADD, uintptr(unsafe.Pointer(&tm.nid)))

	return tm, nil
}

func (tm *TrayManager) UpdateStatus(state string, msg string) {
	tip := fmt.Sprintf("NNU-Connect: %s (%s)", state, msg)
	if len(tip) > 120 {
		tip = tip[:120]
	}
	copyToUTF16Array(tm.nid.SzTip[:], tip)
	procShell_NotifyIconW.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&tm.nid)))
}

func (tm *TrayManager) ShowContextMenu() {
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	hMenu, _, _ := procCreatePopupMenu.Call()
	if hMenu == 0 {
		return
	}
	defer procDestroyMenu.Call(hMenu)

	procAppendMenuW.Call(hMenu, MF_STRING, uintptr(CMD_SHOW_DASHBOARD), uintptr(unsafe.Pointer(stringToUTF16Ptr("🖥️  打开控制面板"))))
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	procAppendMenuW.Call(hMenu, MF_STRING, uintptr(CMD_CONNECT), uintptr(unsafe.Pointer(stringToUTF16Ptr("🚀 立即连接 VPN"))))
	procAppendMenuW.Call(hMenu, MF_STRING, uintptr(CMD_DISCONNECT), uintptr(unsafe.Pointer(stringToUTF16Ptr("⏹️  断开连接"))))
	procAppendMenuW.Call(hMenu, MF_SEPARATOR, 0, 0)
	procAppendMenuW.Call(hMenu, MF_STRING, uintptr(CMD_EXIT), uintptr(unsafe.Pointer(stringToUTF16Ptr("❌ 退出 NNU-Connect"))))

	procSetForegroundWindow.Call(tm.hwnd)
	procTrackPopupMenu.Call(hMenu, TPM_RIGHTALIGN|TPM_BOTTOMALIGN, uintptr(pt.X), uintptr(pt.Y), 0, tm.hwnd, 0)
}

func (tm *TrayManager) OpenDashboard() {
	// Try opening Edge in App mode (standalone window without browser frame)
	edgeCmd := exec.Command("cmd", "/c", "start", "msedge", fmt.Sprintf("--app=%s", tm.serverURL))
	if err := edgeCmd.Start(); err != nil {
		// Fallback to default browser
		_ = exec.Command("cmd", "/c", "start", tm.serverURL).Start()
	}
}

func (tm *TrayManager) RemoveIcon() {
	procShell_NotifyIconW.Call(NIM_DELETE, uintptr(unsafe.Pointer(&tm.nid)))
}

func (tm *TrayManager) RunMessageLoop() {
	type MSG struct {
		HWnd    uintptr
		Message uint32
		WParam  uintptr
		LParam  uintptr
		Time    uint32
		Pt      POINT
	}

	var msg MSG
	for {
		r, _, _ := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

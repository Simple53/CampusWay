//go:build windows

package main

import (
	"errors"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const cryptProtectUIForbidden = 0x1

func canRemember() bool { return true }

type dataBlob struct {
	size uint32
	data *byte
}

var (
	crypt32            = windows.NewLazySystemDLL("crypt32.dll")
	cryptProtectData   = crypt32.NewProc("CryptProtectData")
	cryptUnprotectData = crypt32.NewProc("CryptUnprotectData")
)

func makeBlob(data []byte) dataBlob {
	if len(data) == 0 {
		return dataBlob{}
	}
	return dataBlob{size: uint32(len(data)), data: &data[0]}
}

func protectCredential(plain []byte) ([]byte, error) {
	in := makeBlob(plain)
	var out dataBlob
	result, _, callErr := cryptProtectData.Call(
		uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0,
		cryptProtectUIForbidden, uintptr(unsafe.Pointer(&out)),
	)
	if result == 0 {
		if callErr == syscall.Errno(0) {
			callErr = errors.New("CryptProtectData failed")
		}
		return nil, callErr
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.data))))
	return append([]byte(nil), unsafe.Slice(out.data, out.size)...), nil
}

func unprotectCredential(ciphertext []byte) ([]byte, error) {
	in := makeBlob(ciphertext)
	var out dataBlob
	result, _, callErr := cryptUnprotectData.Call(
		uintptr(unsafe.Pointer(&in)), 0, 0, 0, 0,
		cryptProtectUIForbidden, uintptr(unsafe.Pointer(&out)),
	)
	if result == 0 {
		if callErr == syscall.Errno(0) {
			callErr = errors.New("CryptUnprotectData failed")
		}
		return nil, callErr
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.data))))
	return append([]byte(nil), unsafe.Slice(out.data, out.size)...), nil
}

func saveCredential(path, password string) error {
	protected, err := protectCredential([]byte(password))
	if err != nil {
		return err
	}
	return os.WriteFile(path, protected, 0600)
}

func loadCredential(path string) (string, error) {
	protected, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	plain, err := unprotectCredential(protected)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func deleteCredential(path string) error {
	return os.Remove(path)
}

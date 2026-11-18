//go:build windows

package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"debug/pe"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const updateDirPrefix = ".routedesk-update-"
const maxUpdateSize = 64 << 20

func fetchUpdateArchive(asset releaseAsset) ([]byte, error) {
	u, err := url.Parse(asset.URL)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" || u.User != nil ||
		!strings.HasPrefix(u.Path, "/"+defaultUpdateRepo+"/releases/download/") || filepath.Base(u.Path) != asset.Name {
		return nil, errors.New("Unexpected GitHub asset URL")
	}
	digest := strings.TrimPrefix(asset.Digest, "sha256:")
	sum, err := hex.DecodeString(digest)
	if err != nil || len(sum) != sha256.Size || asset.Size <= 0 || asset.Size > maxUpdateSize {
		return nil, errors.New("Update checksum or size is unavailable")
	}
	client := http.Client{Timeout: 3 * time.Minute, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 || req.URL.Scheme != "https" || !oneOf(req.URL.Host, "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com") {
			return errors.New("Unexpected GitHub asset redirect")
		}
		return nil
	}}
	resp, err := client.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, asset.Size+1))
	if err != nil {
		return nil, err
	}
	actual := sha256.Sum256(data)
	if int64(len(data)) != asset.Size || !bytes.Equal(actual[:], sum) {
		return nil, errors.New("Update checksum did not match")
	}
	return data, nil
}

func extractUpdateExecutable(data []byte, path string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	var executable *zip.File
	for _, entry := range reader.File {
		if entry.Name == "RouteDesk.exe" {
			if executable != nil || !entry.Mode().IsRegular() || entry.UncompressedSize64 > maxUpdateSize {
				return errors.New("Invalid desktop executable in update")
			}
			executable = entry
		}
	}
	if executable == nil {
		return errors.New("RouteDesk.exe is missing from the update")
	}
	input, err := executable.Open()
	if err != nil {
		return err
	}
	defer input.Close()
	out, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(out, io.LimitReader(input, maxUpdateSize+1))
	closeErr := out.Close()
	if err := errors.Join(copyErr, closeErr); err != nil {
		return err
	}
	if uint64(n) != executable.UncompressedSize64 {
		return errors.New("Invalid executable size")
	}
	image, err := pe.Open(path)
	if err != nil {
		return err
	}
	defer image.Close()
	if image.FileHeader.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || image.OptionalHeader == nil {
		return errors.New("Update is not a Windows x64 application")
	}
	return nil
}

func prepareUpdate(asset releaseAsset) (install func() error, discard func(), err error) {
	target, err := os.Executable()
	if err != nil {
		return nil, nil, err
	}
	stage, err := os.MkdirTemp(filepath.Dir(target), updateDirPrefix)
	if err != nil {
		return nil, nil, err
	}
	discard = func() { removeUpdateFiles(stage) }
	ok := false
	defer func() {
		if !ok {
			discard()
		}
	}()
	data, err := fetchUpdateArchive(asset)
	if err != nil {
		return nil, nil, err
	}
	if err = extractUpdateExecutable(data, filepath.Join(stage, "new.exe")); err != nil {
		return nil, nil, err
	}
	content, err := os.ReadFile(target)
	if err != nil {
		return nil, nil, err
	}
	helper := filepath.Join(stage, "installer.exe")
	if err = os.WriteFile(helper, content, 0700); err != nil {
		return nil, nil, err
	}
	ok = true
	return func() error {
		cmd := exec.Command(helper, "--install-update", target, strconv.Itoa(os.Getpid()))
		if err := cmd.Start(); err != nil {
			return err
		}
		_ = cmd.Process.Release()
		return nil
	}, discard, nil
}

func removeUpdateFiles(stage string) {
	for _, name := range []string{"installer.exe", "previous.exe", "new.exe"} {
		_ = os.Remove(filepath.Join(stage, name))
	}
	_ = os.Remove(stage)
}

func handleUpdateCommand(args []string) bool {
	if len(args) < 2 {
		return false
	}
	if args[1] == "--cleanup-update" && len(args) == 3 {
		exe, err := os.Executable()
		stage := filepath.Clean(args[2])
		if err == nil && filepath.Dir(stage) == filepath.Dir(exe) && strings.HasPrefix(filepath.Base(stage), updateDirPrefix) {
			go func() {
				for i := 0; i < 5; i++ {
					removeUpdateFiles(stage)
					if _, err := os.Stat(stage); os.IsNotExist(err) {
						break
					}
					time.Sleep(time.Second)
				}
			}()
		}
		return false
	}
	if args[1] != "--install-update" {
		return false
	}
	if len(args) != 4 {
		return true
	}
	me, err := os.Executable()
	target := filepath.Clean(args[2])
	stage := filepath.Dir(me)
	pid, pidErr := strconv.ParseUint(args[3], 10, 32)
	if err != nil || pidErr != nil || pid == 0 || filepath.Base(me) != "installer.exe" ||
		!strings.HasPrefix(filepath.Base(stage), updateDirPrefix) || filepath.Dir(stage) != filepath.Dir(target) ||
		!strings.HasSuffix(strings.ToLower(target), ".exe") {
		return true
	}
	if err := installStagedUpdate(stage, target, uint32(pid)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		message, _ := windows.UTF16PtrFromString(err.Error())
		title, _ := windows.UTF16PtrFromString("RouteDesk update")
		windows.MessageBox(0, message, title, windows.MB_OK|windows.MB_ICONERROR)
	}
	return true
}

func installStagedUpdate(stage, target string, pid uint32) error {
	parent, err := windows.OpenProcess(windows.SYNCHRONIZE, false, pid)
	if err == nil {
		defer windows.CloseHandle(parent)
		if status, err := windows.WaitForSingleObject(parent, 30000); err != nil || status != windows.WAIT_OBJECT_0 {
			return errors.New("RouteDesk did not exit for the update")
		}
	} else if !errors.Is(err, windows.ERROR_INVALID_PARAMETER) {
		return err
	}
	backup := filepath.Join(stage, "previous.exe")
	if err = replaceUpdateFile(target, filepath.Join(stage, "new.exe"), backup); err != nil {
		if _, statErr := os.Stat(target); os.IsNotExist(statErr) {
			_ = os.Rename(backup, target)
		}
		_ = exec.Command(target).Start()
		return fmt.Errorf("Replace RouteDesk: %w", err)
	}
	cmd := exec.Command(target, "--cleanup-update", stage)
	if err = cmd.Start(); err != nil {
		moveErr := replaceUpdateFile(target, backup, "")
		if moveErr == nil {
			_ = exec.Command(target).Start()
		}
		return errors.Join(err, moveErr)
	}
	_ = cmd.Process.Release()
	return nil
}

func replaceUpdateFile(target, replacement, backup string) error {
	from, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	to, err := windows.UTF16PtrFromString(replacement)
	if err != nil {
		return err
	}
	var saved *uint16
	if backup != "" {
		saved, err = windows.UTF16PtrFromString(backup)
		if err != nil {
			return err
		}
	}
	replace := windows.NewLazySystemDLL("kernel32.dll").NewProc("ReplaceFileW")
	ok, _, callErr := replace.Call(uintptr(unsafe.Pointer(from)), uintptr(unsafe.Pointer(to)), uintptr(unsafe.Pointer(saved)), 0, 0, 0)
	if ok == 0 {
		return callErr
	}
	return nil
}

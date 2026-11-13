//go:build !windows

package main

import (
	"errors"
	"os"
)

// Never fall back to plaintext when an OS credential backend is unavailable.
func canRemember() bool { return false }
func saveCredential(path, password string) error {
	return errors.New("Secure password storage is unavailable on this platform")
}
func loadCredential(path string) (string, error) { return "", os.ErrNotExist }
func deleteCredential(path string) error         { return os.Remove(path) }

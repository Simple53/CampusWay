//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCredentialRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credential.bin")
	const password = "nnu-test-password"

	if err := saveCredential(path, password); err != nil {
		t.Fatal(err)
	}
	ciphertext, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(ciphertext) == password {
		t.Fatal("credential was written as plaintext")
	}
	got, err := loadCredential(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != password {
		t.Fatalf("password = %q, want %q", got, password)
	}
	if err := deleteCredential(path); err != nil {
		t.Fatal(err)
	}
}

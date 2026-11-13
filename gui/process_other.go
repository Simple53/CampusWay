//go:build !windows

package main

import "os/exec"

func hideChild(cmd *exec.Cmd) {}

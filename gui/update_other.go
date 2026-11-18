//go:build !windows

package main

import "errors"

func prepareUpdate(releaseAsset) (func() error, func(), error) {
	return nil, nil, errors.New("This release requires a manual installation")
}

func handleUpdateCommand([]string) bool { return false }

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const defaultUpdateRepo = "Simple53/CampusWay"

var updateRepoPattern = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

type releaseAsset struct {
	Name   string `json:"name"`
	URL    string `json:"browser_download_url"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

type githubRelease struct {
	Tag    string         `json:"tag_name"`
	Assets []releaseAsset `json:"assets"`
}

func fetchRelease(repository string) (githubRelease, error) {
	var release githubRelease
	if !updateRepoPattern.MatchString(repository) {
		return release, errors.New("Invalid GitHub repository")
	}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+repository+"/releases/latest", nil)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "RouteDesk")
	h := http.Client{Timeout: 10 * time.Second}
	resp, err := h.Do(req)
	if err != nil {
		return release, errors.New("Unable to check for updates")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return release, errors.New("No published release found")
	}
	if resp.StatusCode != http.StatusOK {
		return release, errors.New("Unable to check for updates")
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&release); err != nil {
		return release, errors.New("Invalid release response")
	}
	return release, nil
}

func desktopUpdateAsset(release githubRelease) (releaseAsset, bool) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return releaseAsset{}, false
	}
	name := "RouteDesk-" + strings.TrimLeft(release.Tag, "vV") + "-windows-x64-portable.zip"
	for _, asset := range release.Assets {
		if asset.Name == name {
			return asset, true
		}
	}
	return releaseAsset{}, false
}

func (s *AppServer) CheckVersion(repository string) (map[string]string, error) {
	if repository == "" {
		repository = defaultUpdateRepo
	}
	release, err := fetchRelease(repository)
	if err != nil {
		return nil, err
	}
	result := map[string]string{"current": version, "latest": release.Tag, "repository": repository, "status": releaseStatus(version, release.Tag)}
	if _, ok := desktopUpdateAsset(release); ok && repository == defaultUpdateRepo {
		result["install"] = "true"
	}
	return result, nil
}

func (s *AppServer) ApplyUpdate(repository string) (map[string]string, error) {
	if repository != "" && repository != defaultUpdateRepo {
		return nil, errors.New("Install updates from the official RouteDesk repository")
	}
	if !s.updateMu.TryLock() {
		return nil, errors.New("An update is already in progress")
	}
	defer s.updateMu.Unlock()
	release, err := fetchRelease(defaultUpdateRepo)
	if err != nil {
		return nil, err
	}
	if releaseStatus(version, release.Tag) != "A different release is available" {
		return nil, errors.New("No newer release is available")
	}
	asset, ok := desktopUpdateAsset(release)
	if !ok || s.quitForUpdate == nil {
		return nil, errors.New("This release requires a manual installation")
	}
	install, discard, err := prepareUpdate(asset)
	if err != nil {
		return nil, fmt.Errorf("Prepare update: %w", err)
	}
	if !s.operation.TryLock() {
		discard()
		return nil, errors.New("Wait for the current connection operation")
	}
	stopErr := s.stop()
	s.operation.Unlock()
	if stopErr != nil {
		discard()
		return nil, stopErr
	}
	if err := install(); err != nil {
		discard()
		return nil, err
	}
	s.quitForUpdate()
	return map[string]string{"version": release.Tag, "status": "restarting"}, nil
}

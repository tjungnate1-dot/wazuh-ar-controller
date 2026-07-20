package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// blockIP adds an IP to the configured response backend.
func blockIP(ip string, cfg *Config) (string, error) {
	switch cfg.BlockMethod {
	case "mock":
		return blockIPMock(ip, cfg.Mock.StatePath)

	case "ipset":
		return blockIPSet(ip, cfg.IPSet.SetName)

	default:
		return "", fmt.Errorf(
			"unsupported block method %q",
			cfg.BlockMethod,
		)
	}
}

// blockIPMock simulates blocking an IP.
//
// saves the IP into a JSON file.
// testing cuz this is a mac <3
func blockIPMock(ip string, statePath string) (string, error) {
	blockedIPs, err := loadMockState(statePath)
	if err != nil {
		return "", fmt.Errorf("load mock block state: %w", err)
	}

	// Check whether the IP is already present
	if blockedIPs[ip] {
		return "already_blocked", nil
	}

	// Add the IP to the in-memory map
	blockedIPs[ip] = true

	// Save the updated map back to disk
	if err := saveMockState(statePath, blockedIPs); err != nil {
		return "", fmt.Errorf("save mock block state: %w", err)
	}

	return "added", nil
}

// blockIPSet adds an IP to the Linux ipset
//
// this function is the actual thing, use on the VM
func blockIPSet(ip string, setName string) (string, error) {
	exists, err := ipSetContains(setName, ip)
	if err != nil {
		return "", err
	}

	if exists {
		return "already_blocked", nil
	}

	// Do not use "sh -c".
	// Passing each argument separately avoids shell injection.
	cmd := exec.Command(
		"ipset",
		"add",
		setName,
		ip,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(
			"ipset add failed: %w; output=%q",
			err,
			strings.TrimSpace(string(output)),
		)
	}

	return "added", nil
}

// ipSetContains checks whether an IP exists in the ipset.
//
// ipset test normally returns:
// exit code 0: IP exists
// exit code 1: IP does not exist
func ipSetContains(setName string, ip string) (bool, error) {
	cmd := exec.Command(
		"ipset",
		"test",
		setName,
		ip,
	)

	output, err := cmd.CombinedOutput()

	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}

	return false, fmt.Errorf(
		"ipset test failed: %w; output=%q",
		err,
		strings.TrimSpace(string(output)),
	)
}

// loadMockState reads the mock block list from disk.
//
// IP : true
//
// If the file does not exist yet, an empty block list is returned.
func loadMockState(path string) (map[string]bool, error) {
	blockedIPs := make(map[string]bool)

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return blockedIPs, nil
		}

		return nil, fmt.Errorf(
			"read mock state file %q: %w",
			path,
			err,
		)
	}

	// An empty state file is treated as an empty block list.
	if len(strings.TrimSpace(string(content))) == 0 {
		return blockedIPs, nil
	}

	if err := json.Unmarshal(content, &blockedIPs); err != nil {
		return nil, fmt.Errorf(
			"decode mock state file %q: %w",
			path,
			err,
		)
	}

	return blockedIPs, nil
}

// saveMockState safely writes the mock block list to disk.
//
// It first writes to a temporary file and then renames it.
// This reduces the chance of leaving a partially written file.
func saveMockState(
	path string,
	blockedIPs map[string]bool,
) error {
	content, err := json.MarshalIndent(blockedIPs, "", "  ")
	if err != nil {
		return fmt.Errorf("encode mock state: %w", err)
	}

	directory := filepath.Dir(path)

	if directory != "." {
		if err := os.MkdirAll(directory, 0750); err != nil {
			return fmt.Errorf(
				"create mock state directory %q: %w",
				directory,
				err,
			)
		}
	}

	temporaryPath := path + ".tmp"

	if err := os.WriteFile(temporaryPath, content, 0600); err != nil {
		return fmt.Errorf(
			"write temporary mock state %q: %w",
			temporaryPath,
			err,
		)
	}

	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf(
			"replace mock state file %q: %w",
			path,
			err,
		)
	}

	return nil
}

package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// blockIP adds an IP to the configured ipset.
func blockIP(ip string, cfg *Config) (string, error) {
	return blockIPSet(ip, cfg.IPSet.SetName)
}

// blockIPSet adds an IP to the Linux ipset.
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
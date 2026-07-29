package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// flushIP removes one IP from the configured ipset.
func flushIP(ip string, cfg *Config) (string, error) {
	return flushIPSet(ip, cfg.IPSet.SetName)
}

// flushIPSet removes one IP from a real Linux ipset.
func flushIPSet(ip string, setName string) (string, error) {
	exists, err := ipSetContains(setName, ip)
	if err != nil {
		return "", err
	}

	if !exists {
		return "not_found", nil
	}

	cmd := exec.Command(
		"ipset",
		"del",
		setName,
		ip,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf(
			"ipset delete failed: %w; output=%q",
			err,
			strings.TrimSpace(string(output)),
		)
	}

	return "deleted", nil
}
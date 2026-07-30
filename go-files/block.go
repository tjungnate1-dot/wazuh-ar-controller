package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// blockIP adds an IP to the configured ipset.
func blockIP(ip string, cfg *Config) (string, error) {
	//validate
	exists, err := ipSetContains(cfg.IPSet.SetName, ip)
	if err != nil {
		return "", err
	}

	if exists {
		return "already_blocked", nil
	}

	cmd := exec.Command("ipset", "add", cfg.IPSet.SetName, ip, )

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ipset add failed: %w; output=%q", err, strings.TrimSpace(string(output)), )
	}

	return "added", nil
}


func ipSetContains(setName string, ip string) (bool, error) {
	//runs this command
	cmd := exec.Command("ipset", "test", setName, ip, )
	output, err := cmd.CombinedOutput()

	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}

	return false, fmt.Errorf("ipset test failed: %w; output=%q",
	 err, strings.TrimSpace(string(output)), )
}
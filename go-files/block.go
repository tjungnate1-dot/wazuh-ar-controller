package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// blockIP adds an IP to the configured ipset.
func blockIP(ip string, cfg *Config) (string, error) {
	// make sure ipset and table exists
	set_name := cfg.IPSet.SetName
	err := createFirewall(set_name)
	if err != nil {
		return "", err
	}
	// Check whether the IP is already present.
	exists, err := ipSetContains(ip, set_name)
	if err != nil {
		return "", err
	}

	if exists {
		return "already_blocked", nil
	}

	// Add the IP to the set.
	cmd := exec.Command("ipset", "add", set_name, ip, "-exist")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ipset add failed: %w; output=%q", err, strings.TrimSpace(string(output)))
	}

	return "added", nil
}

func checkIPSet(set_name string) error {
	cmd := exec.Command("ipset", "create", set_name, "hash:ip", "-exist")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create ipset: %v; output=%q", err, output)
	}

	return nil
}

func createFirewall(set_name string) error {
	if err := checkIPSet(set_name); err != nil {
		return err
	}

	check := exec.Command(
		"iptables", "-C", "INPUT", "-m", "set", "--match-set",
		set_name, "src", "-j", "DROP",
	)

	if err := check.Run(); err == nil {
		return nil // Rule already exists
	}

	add := exec.Command(
		"iptables", "-I", "INPUT", "1", "-m", "set",
		"--match-set", set_name, "src", "-j", "DROP",
	)

	output, err := add.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create iptables rule: %v; output=%q", err, output)
	}

	return nil
}

func ipSetContains(ip string, set_name string) (bool, error) {
	//runs this command
	cmd := exec.Command("ipset", "test", set_name, ip)
	output, err := cmd.CombinedOutput()

	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return false, nil
	}

	return false, fmt.Errorf("ipset test failed: %w; output=%q",
		err, strings.TrimSpace(string(output)))
}

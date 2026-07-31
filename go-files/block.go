package main

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// blockIP adds an IP to the configured ipset.
func blockIP(ip string) error {
    if err := ensureFirewallReady(); err != nil {
        return err
    }

    cmd := exec.Command("ipset", "add", "wazuh-ar-blocklist", ip, "-exist",)

    output, err := cmd.CombinedOutput()
    if err != nil {
        return fmt.Errorf("ipset add failed: %v; output=%q", err, output)
    }

    return nil
}

func checkIPSet() error{
	cmd := exec.Command("ipset", "create", "wazuh-ar-blocklist", "hash:ip", "-exist", )
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt. Errorf("failed to create ipset: %v; output=%q", err, output)
	}

	return nil
}

func ensureFirewallReady() error {
    if err := checkIPSet(); err != nil {
        return err
    }

    check := exec.Command(
        "iptables", "-C", "INPUT", "-m", "set", "--match-set", 
		"wazuh-ar-blocklist", "src", "-j", "DROP",
	)

    if err := check.Run(); err == nil {
        return nil // Rule already exists
    }

    add := exec.Command(
        "iptables", "-I", "INPUT", "1", "-m", "set", 
		"--match-set", "wazuh-ar-blocklist", "src", "-j", "DROP",
    )

    output, err := add.CombinedOutput()
    if err != nil {
        return fmt.Errorf("failed to create iptables rule: %v; output=%q", err, output)
    }

    return nil
}

func ipSetContains(ip string) (bool, error) {
	//runs this command
	cmd := exec.Command("ipset", "test", "wazuh-ar-blocklist", ip, )
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
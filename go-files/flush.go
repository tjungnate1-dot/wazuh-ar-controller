package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// flushIP removes one IP from the configured ipset.
func flushIP(ip string) (string, error) {
	exists, err := ipSetContains(ip)
	cmd := exec.Command("ipset", "del", "wazuh-ar-blocklist", ip)

	output, err := cmd.CombinedOutput()

	if !exists {
		return "not found", nil
	}
	if err != nil {
		return "", fmt.Errorf("ipset delete failed: %w; output=%q", err, strings.TrimSpace(string(output)))
	}

	return "deleted", nil
}

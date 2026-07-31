package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// flushIP removes one IP from the configured ipset.
func flushIP(ip string) (string, error) {
	exists, err := ipSetContains(ip)
	if err != nil {
		return "", err
	}

	if !exists {
		return "not_found", nil
	}

	cmd := exec.Command("ipset", "del","wazuh-ar-blocklist", ip, )

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ipset delete failed: %w; output=%q", err, strings.TrimSpace(string(output)), )
	}

	return "deleted", nil
}

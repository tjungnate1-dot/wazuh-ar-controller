package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// flushIP removes one IP from the configured ipset.
func flushIP(ip string, cfg *Config) (string, error) {
	set_name := cfg.IPSet.SetName
	exists, err := ipSetContains(ip, set_name)
	cmd := exec.Command("ipset", "del", set_name, ip)

	output, err := cmd.CombinedOutput()

	if !exists {
		return "not found", nil
	}
	if err != nil {
		return "", fmt.Errorf("ipset delete failed: %w; output=%q", err, strings.TrimSpace(string(output)))
	}

	return "deleted", nil
}

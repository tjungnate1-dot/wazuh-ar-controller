package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// flushIP removes one IP

func flushIP(ip string, cfg *Config) (string, error) {
	switch cfg.BlockMethod {
	case "mock":
		return flushIPMock(ip, cfg.Mock.StatePath)
	case "ipset":
		return flushIPSet(ip, cfg.IPSet.SetName)
	default:
		return "", fmt.Errorf(
			"unsupported block method %q",
			cfg.BlockMethod,
		)
	}
}

// flushIPMock removes one IP from file Can delete on VM 
func flushIPMock(ip string, statePath string) (string, error) {
	blockedIPs, err := loadMockState(statePath)
	if err != nil {
		return "", fmt.Errorf("load mock block state: %w", err)
	}

	//if the IP is not in the map, there is nothing to remove.
	if !blockedIPs[ip] {
		return "not_found", nil
	}

	//remove only the requested IP.
	delete(blockedIPs, ip)

	if err := saveMockState(statePath, blockedIPs); err != nil {
		return "", fmt.Errorf("save mock block state: %w", err)
	}

	return "deleted", nil
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

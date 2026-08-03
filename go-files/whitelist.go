package main

import (
	"fmt"
	"net/netip"
	"strings"
)

// isWhitelisted checks whether the source IP matches any
// exact IP or CIDR entry in the configured whitelist.

func isWhitelisted(ipText string, whitelist []string,) (bool, string, error) {
	// Parse and validate the source IP first.
	sourceIP, err := netip.ParseAddr(strings.TrimSpace(ipText),)
	if err != nil {
		return false, "", fmt.Errorf("invalid source IP %q: %w", ipText, err,)
	}

	for _, rawEntry := range whitelist {
		entry := strings.TrimSpace(rawEntry)

		if entry == "" {
			return false, "", fmt.Errorf(
				"whitelist contains an empty entry",
			)
		}

		// treat as an exact IP first
		allowedIP, ipErr := netip.ParseAddr(entry)
		if ipErr == nil {
			if sourceIP == allowedIP {
				return true, entry, nil
			}

			continue
		}

		// If it is not an exact IP, try treating it as CIDR.
		//
		// Example:
		// 192.168.0.0/16
		prefix, prefixErr := netip.ParsePrefix(entry)
		if prefixErr != nil {
			return false, "", fmt.Errorf(
				"invalid whitelist entry %q: expected IP or CIDR",
				entry,
			)
		}

		if prefix.Contains(sourceIP) {
			return true, entry, nil
		}
	}

	return false, "", nil
}

//straight from ChatGPT, should check first 
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// ControlResponse represents Wazuh's answer after we send check_keys.
//
// Expected command values:
//
//	continue
//	abort
type ControlResponse struct {
	Version    int               `json:"version"`
	Origin     Origin            `json:"origin"`
	Command    string            `json:"command"`
	Parameters map[string]any    `json:"parameters"`
}

// Origin identifies which Wazuh component sent the message.
type Origin struct {
	Name   string `json:"name"`
	Module string `json:"module"`
}

// CheckKeysMessage is sent by our Active Response binary to Wazuh.
//
// Wazuh uses the keys to determine whether the same response
// is already active and should therefore be aborted.
type CheckKeysMessage struct {
	Version    int                 `json:"version"`
	Origin     Origin              `json:"origin"`
	Command    string              `json:"command"`
	Parameters CheckKeysParameters `json:"parameters"`
}

// CheckKeysParameters contains the values that uniquely identify
// this Active Response action.
//
// For IP blocking, the source IP is the key.
type CheckKeysParameters struct {
	Keys []string `json:"keys"`
}

// readJSONLine reads exactly one newline-terminated JSON message.
//
// Wazuh Active Response uses a line-based protocol. This function
// must not attempt to read until EOF because Wazuh may send another
// message later during the same process execution.
func readJSONLine(
	reader *bufio.Reader,
	target any,
) error {
	line, err := reader.ReadString('\n')
	if err != nil {
		// During manual testing, a file may end without a final newline.
		// Accept the remaining content when EOF is reached.
		if err != io.EOF || strings.TrimSpace(line) == "" {
			return fmt.Errorf("read JSON line from stdin: %w", err)
		}
	}

	line = strings.TrimSpace(line)

	if line == "" {
		return fmt.Errorf("received an empty JSON message")
	}

	if err := json.Unmarshal([]byte(line), target); err != nil {
		return fmt.Errorf("decode JSON message: %w", err)
	}

	return nil
}

// writeJSONLine writes one JSON object followed by a newline.
//
// The newline and Flush call are mandatory for the Wazuh protocol.
// Without them, Wazuh and the response script may wait forever
// for each other.
func writeJSONLine(
	writer *bufio.Writer,
	message any,
) error {
	content, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode JSON message: %w", err)
	}

	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("write JSON message: %w", err)
	}

	if err := writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("write JSON newline: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush JSON message: %w", err)
	}

	return nil
}

// requestWazuhKeyCheck asks Wazuh whether this source IP
// should continue into the block operation.
//
// Return values:
//
// true, nil
//     Wazuh replied "continue".
//
// false, nil
//     Wazuh replied "abort".
//
// false, error
//     An invalid or unreadable response was received.
func requestWazuhKeyCheck(
	reader *bufio.Reader,
	writer *bufio.Writer,
	programName string,
	sourceIP string,
) (bool, error) {
	request := CheckKeysMessage{
		Version: 1,
		Origin: Origin{
			Name:   programName,
			Module: "active-response",
		},
		Command: "check_keys",
		Parameters: CheckKeysParameters{
			Keys: []string{sourceIP},
		},
	}

	// Send check_keys to Wazuh through stdout.
	if err := writeJSONLine(writer, request); err != nil {
		return false, fmt.Errorf("send check_keys: %w", err)
	}

	// Wazuh now sends either continue or abort through stdin.
	var response ControlResponse

	if err := readJSONLine(reader, &response); err != nil {
		return false, fmt.Errorf(
			"read check_keys response: %w",
			err,
		)
	}

	switch response.Command {
	case "continue":
		return true, nil

	case "abort":
		return false, nil

	default:
		return false, fmt.Errorf(
			"unexpected Wazuh control command %q",
			response.Command,
		)
	}
}
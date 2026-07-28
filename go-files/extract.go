

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"strconv"
	"strings"
)

type Result struct {
	Extracted map[string]any `json:"extracted"`
	Original  map[string]any `json:"original,omitempty"`
}

// decodeJSON reads one JSON object and converts it into a map
func decodeJSON(reader io.Reader) (map[string]any, error) {
	decoder := json.NewDecoder(reader)
	decoder.UseNumber()

	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}

	var extra any
	err := decoder.Decode(&extra)
	if err == io.EOF {
		return document, nil
	}
	if err == nil {
		return nil, errors.New("multiple JSON documents detected")
	}
	return nil, fmt.Errorf("unexpected trailing content: %w", err)
}

// extractFields follows the JSON paths from configuration, validates values, and builds the final result
func extractFields(document map[string]any, cfg *Config) (*Result, error) {
	result := &Result{
		Extracted: make(map[string]any),
	}

	if cfg.PrintOriginal {
		result.Original = document
	}

	for outputName, jsonPath := range cfg.Fields {
		value, found := getByPath(document, jsonPath)
		if !found {
			result.Extracted[outputName] = nil
			continue
		}
		result.Extracted[outputName] = value
	}

	for _, requiredField := range cfg.Required {
		value, exists := result.Extracted[requiredField]
		if !exists || isEmpty(value) {
			path := cfg.Fields[requiredField]
			return nil, fmt.Errorf("required field %q was not found or was empty at JSON path %q", requiredField, path)
		}
	}

	for _, fieldName := range cfg.ValidateIP {
		value := result.Extracted[fieldName]
		ipText, ok := value.(string)
		if !ok {
			return nil, fmt.Errorf("field %q must contain a string IP address, got %T", fieldName, value)
		}
		if _, err := netip.ParseAddr(ipText); err != nil {
			return nil, fmt.Errorf("field %q contains invalid IP address %q: %w", fieldName, ipText, err)
		}
	}

	return result, nil
}

func getByPath(document any, path string) (any, bool) {
	current := document

	for _, segment := range strings.Split(path, ".") {
		if segment == "" {
			return nil, false
		}

		switch typed := current.(type) {
		case map[string]any:
			next, exists := typed[segment]
			if !exists {
				return nil, false
			}
			current = next
		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil {
				return nil, false
			}
			if index < 0 || index >= len(typed) {
				return nil, false
			}
			current = typed[index]
		default:
			return nil, false
		}
	}
	return current, true
}

func isEmpty(value any) bool {
	if value == nil {
		return true
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed) == ""
	case []any:
		return len(typed) == 0
	case map[string]any:
		return len(typed) == 0
	default:
		return false
	}
}

func getStringField(fields map[string]any, name string) (string, error) {
	value, exists := fields[name]
	if !exists {
		return "", fmt.Errorf("field %q was not found", name)
	}
	if value == nil {
		return "", fmt.Errorf("field %q is null", name)
	}

	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("field %q must be a string, got %T", name, value)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return "", fmt.Errorf("field %q cannot be empty", name)
	}
	return text, nil
}

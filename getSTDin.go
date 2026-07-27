package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

//where file is stored
const outputFilePath = "/Users/nate/ARC/wazuh-ar/test-stdin.json"

func main() {
	// 1. Read stdin
	rawInput, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
		os.Exit(1)
	}

	if len(bytes.TrimSpace(rawInput)) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No data received on stdin.")
		os.Exit(1)
	}

	// 2. Validate JSON structure and format with 2-space indentation
	var formattedJSON bytes.Buffer
	if err := json.Indent(&formattedJSON, rawInput, "", "  "); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Invalid JSON input: %v\n", err)
		os.Exit(1)
	}

	// 3. Ensure the target directory exists before writing
	outputDir := filepath.Dir(outputFilePath)
	if outputDir != "." && outputDir != "" {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", outputDir, err)
			os.Exit(1)
		}
	}

	// 4. Write the formatted JSON payload to disk
	// Permissions set to 0644 (read/write for owner, read-only for others)
	err = os.WriteFile(outputFilePath, formattedJSON.Bytes(), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "File I/O error writing to %s: %v\n", outputFilePath, err)
		os.Exit(1)
	}

	fmt.Printf("Successfully written stdin payload to %s\n", outputFilePath)
}
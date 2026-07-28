package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

//output file storage path
const outputFilePath = "./test-stdin.json"

//structs matching wazuh json 
type AlertData struct {
	SrcIP string `json:"srcip"`
}
type Alert struct {
	Data AlertData `json:"data"`
}
type Parameters struct {
	Alert Alert `json:"alert"`
}
type WazuhInput struct {
	Version    int        `json:"version"`
	Command    string     `json:"command"`
	Parameters Parameters `json:"parameters"`
}
type ReplyMessage struct {
	Version int `json:"version"`
	Origin  struct {
		Name   string `json:"name"`
		Module string `json:"module"`
	} `json:"origin"`
	Command    string `json:"command"`
	Parameters struct {
		Keys []string `json:"keys"`
	} `json:"parameters"`
}

func main() {
	//create a line scanner on STDIN to avoid hanging/deadlocks
	scanner := bufio.NewScanner(os.Stdin)

	//1. read the first JSON message from wazu (up to '\n')
	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "Error: No data received on stdin.")
		os.Exit(1)
	}
	rawInput := scanner.Bytes()

	//parse incoming JSON
	var inputPayload WazuhInput
	if err := json.Unmarshal(rawInput, &inputPayload); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON input: %v\n", err)
		os.Exit(1)
	}

	//add vs delete
	switch inputPayload.Command {
	case "add":
	//perform add, go on to check_key
	case "delete":
	//unblock ip and stop
		fmt.Fprintln(os.Stderr, "Received 'delete' command. Reverting action...")
		saveToFile(rawInput)
		return
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", inputPayload.Command)
		os.Exit(1)
	}

	//build & send control message to STDOUT (Must end with \n to prevent deadlock!)
	srcIP := inputPayload.Parameters.Alert.Data.SrcIP
	if srcIP != "" {
		ctrlMsg := ReplyMessage{
			Version: 1,
			Command: "check_keys",
		}
		ctrlMsg.Origin.Name = "custom-active-response"
		ctrlMsg.Origin.Module = "active-response"
		ctrlMsg.Parameters.Keys = []string{srcIP}

		ctrlBytes, _ := json.Marshal(ctrlMsg)

		// adds the '\n'
		fmt.Println(string(ctrlBytes))

		//4. read response thru STDIN
		if scanner.Scan() {
			responseRaw := scanner.Bytes()

			var managerResponse struct {
				Command string `json:"command"`
			}
			if err := json.Unmarshal(responseRaw, &managerResponse); err == nil {
				if managerResponse.Command == "abort" {
					fmt.Fprintln(os.Stderr, "Execution aborted by manager (duplicate process active).")
					os.Exit(0)
				}
			}
		}
	}

	//5. write the original to disk
	saveToFile(rawInput)
}

//helper: format and write json to disk
func saveToFile(rawInput []byte) {
	var formattedJSON bytes.Buffer
	if err := json.Indent(&formattedJSON, rawInput, "", "  "); err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting JSON: %v\n", err)
		os.Exit(1)
	}

	outputDir := filepath.Dir(outputFilePath)
	if outputDir != "." && outputDir != "" {
		_ = os.MkdirAll(outputDir, 0755)
	}

	err := os.WriteFile(outputFilePath, formattedJSON.Bytes(), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "File I/O error writing to %s: %v\n", outputFilePath, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Successfully saved output to %s\n", outputFilePath)
}
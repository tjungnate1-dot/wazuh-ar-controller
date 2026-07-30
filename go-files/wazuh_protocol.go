package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)


type CheckKeyMessage struct{
	Version int `json:"version"`

	Origin struct {
		Name string `json:"name"`
		Module string `json:"module"`
	}  `json:"origin"`

	Command string `json:"command"`

	Parameters struct{
		Keys []string `json:"keys"`
	} `json:"parameters"`
}

type ManagerResponse struct {
	Version int `json:"version"`

	Command string `json:"command"`
}


func readJSONLine(reader *bufio.Reader, target any, ) error {
	rawMessage, err := reader.ReadBytes('\n')

	if err != nil {
		if err != io.EOF || len(bytes.TrimSpace(rawMessage)) == 0 {
			return fmt.Errorf("read JSON message: %w", err, )
		}
	}

	rawMessage = bytes.TrimSpace(rawMessage)

	if len(rawMessage) == 0 {
		return fmt.Errorf("recieved an empty JSON message", )
	}

	if err := json.Unmarshal(rawMessage, target); err != nil {
		return fmt.Errorf("decode JSON message: %w", err, )
	}

	return nil
}

func writeJSONLine(writer *bufio.Writer, message any, ) error {
	content, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("encode protocol message %w", err, )
	}

	if _, err := writer.Write(content); err != nil {
		return fmt.Errorf("write protocol message: %w", err, )
	}
	
	if err := writer.WriteByte('\n'); err != nil {
		return fmt.Errorf("write protocol newline: %w", err, )
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush protocol message: %w", err, )
	}

	return nil
}

func requestWazuhKeyCheck(reader *bufio.Reader, writer * bufio.Writer, sourceIP string,) (bool, error) {
	var request CheckKeyMessage

	request.Version = 1
	request.Origin.Name = "custom-active-response"
	request.Origin.Module = "active-response"
	request.Command = "check_keys"
	request.Parameters.Keys = []string{sourceIP}

	if err := writeJSONLine(writer, request); err != nil {
		return false, fmt.Errorf("send check_keys: %w", err,)
	}

	var response ManagerResponse

	if err := readJSONLine(reader, &response); err != nil {
		return false, fmt.Errorf("read wazuh manager response: %w", err, )
	}

	switch response.Command {
	case "continue":
		return true, nil

	case "abort": 
		return false, nil
	default:
		return false, fmt.Errorf("unexpected Wazuh response command %q", response.Command, )
	}
}

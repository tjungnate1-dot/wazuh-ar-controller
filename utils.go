package main

import (
	"fmt"
	"io"
	"log"
	"os"
)

// exitError stop code execution with exit code 1
func exitError(category string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", category, err)
	os.Exit(1)
}

// isKillSwitchActive check if the emergency stop file exists
func isKillSwitchActive(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("check kill switch %q: %w", path, err)
}

// openInput returns the proper reader based on CLI configuration
func openInput(path string) (io.Reader, func(), error) {
	if path == "" {
		return os.Stdin, func() {}, nil
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, func() {}, fmt.Errorf("open %q: %w", path, err)
	}

	return file, func() {
		_ = file.Close()
	}, nil
}

// createLogger configures a generic file logger
func createLogger(path string) (*log.Logger, *os.File, error) {
	logFile, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_APPEND|os.O_WRONLY,
		0640,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("open log file %q: %w", path, err)
	}

	logger := log.New(
		logFile,
		"",
		log.Ldate|log.Ltime|log.LUTC,
	)

	return logger, logFile, nil
}
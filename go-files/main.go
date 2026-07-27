package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

const killSwitchPath = ".killswitch"

func main() {
	configPath := flag.String("config", "../config.yaml", "path to the YAML decoder configuration")
	jsonPath := flag.String("input", "", "path to a JSON file; when omitted, JSON is read from stdin")
	flag.Parse()

	// 1. check killswitch
	killSwitchActive, err := isKillSwitchActive(killSwitchPath)
	if err != nil {
		exitError("kill-switch error", err)
	}
	if killSwitchActive {
		fmt.Fprintf(os.Stderr, "kill switch active at %s, stopping immediately\n", killSwitchPath)
		return
	}

	// 2. load and validate config.yaml
	cfg, err := loadConfig(*configPath)
	if err != nil {
		exitError("configuration error", err)
	}

	// 3. setup logging
	logger, logFile, err := createLogger(cfg.LogPath)
	if err != nil {
		exitError("logging error", err)
	}
	defer logFile.Close()

	logger.Printf("action=start result=success message=%q", "active response controller started")

	// 4. return if disabled
	if !cfg.Enabled {
		logger.Printf("action=controller_check result=skipped reason=%q", "controller disabled in configuration")
		return
	}

	// 5. stream input data
	input, closeInput, err := openInput(*jsonPath)
	if err != nil {
		exitError("input error", err)
	}
	defer closeInput()

	// 6. decode JSON
	document, err := decodeJSON(input)
	if err != nil {
		logger.Printf("action=json_decode result=failed error=%q", err.Error())
		exitError("JSON decode error", err)
	}
	logger.Printf("action=json_decode result=success")

	// 7. extract and check fields
	result, err := extractFields(document, cfg)
	if err != nil {
		logger.Printf("action=field_extract result=failed error=%q", err.Error())
		exitError("field extraction error", err)
	}

	// 8. get source_ip, command, and rule_id
	sourceIP, err := getStringField(result.Extracted, "source_ip")
	if err != nil {
		logger.Printf("action=source_ip_extract result=failed error=%q", err.Error())
		exitError("source IP error", err)
	}

	command, err := getStringField(result.Extracted, "command")
	if err != nil {
		logger.Printf("action=command_extract result=failed error=%q", err.Error())
		exitError("command error", err)
	}

	ruleID, err := getStringField(result.Extracted, "rule_id")
	if err != nil {
		logger.Printf("action=rule_id_extract result=failed error=%q", err.Error())
		exitError("rule ID error", err)
	}
	//log them
	logger.Printf(
		"action=field_extract result=success command=%q source_ip=%q rule_id=%q",
		command,
		sourceIP,
		ruleID,
	)

	switch command {
	case "add":
		// Before blocking, check whether the IP is trusted.
		whitelisted, matchedEntry, err := isWhitelisted(
			sourceIP,
			cfg.Whitelist,
		)
		if err != nil {
			logger.Printf(
				"action=whitelist_check result=failed source_ip=%q error=%q",
				sourceIP,
				err.Error(),
			)

			exitError("whitelist error", err)
		}

		if whitelisted {
			logger.Printf(
				"action=block result=skipped source_ip=%q rule_id=%q reason=%q whitelist_entry=%q",
				sourceIP,
				ruleID,
				"source IP is whitelisted",
				matchedEntry,
			)

			return
		}

		result, err := blockIP(sourceIP, cfg)
		if err != nil {
			logger.Printf(
				"action=block result=failed source_ip=%q rule_id=%q backend=%q error=%q",
				sourceIP,
				ruleID,
				cfg.BlockMethod,
				err.Error(),
			)

			exitError("block error", err)
		}

		switch result {
		case "added":
			logger.Printf(
				"action=block result=success source_ip=%q rule_id=%q backend=%q",
				sourceIP,
				ruleID,
				cfg.BlockMethod,
			)

		case "already_blocked":
			logger.Printf(
				"action=block result=skipped source_ip=%q rule_id=%q backend=%q reason=%q",
				sourceIP,
				ruleID,
				cfg.BlockMethod,
				"IP already blocked",
			)
		}

	case "delete":
		result, err := flushIP(sourceIP, cfg)
		if err != nil {
			logger.Printf(
				"action=unblock result=failed source_ip=%q rule_id=%q backend=%q error=%q",
				sourceIP,
				ruleID,
				cfg.BlockMethod,
				err.Error(),
			)

			exitError("unblock error", err)
		}

		switch result {
		case "deleted":
			logger.Printf(
				"action=unblock result=success source_ip=%q rule_id=%q backend=%q",
				sourceIP,
				ruleID,
				cfg.BlockMethod,
			)

		case "not_found":
			logger.Printf(
				"action=unblock result=skipped source_ip=%q rule_id=%q backend=%q reason=%q",
				sourceIP,
				ruleID,
				cfg.BlockMethod,
				"IP was not blocked",
			)
		}

	default:
		err := fmt.Errorf(
			"unsupported command %q",
			command,
		)

		logger.Printf(
			"action=command_check result=failed command=%q source_ip=%q error=%q",
			command,
			sourceIP,
			err.Error(),
		)

		exitError("command error", err)
	}

	// 9. output results
	output, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		exitError("output encoding error", err)
	}

	//fmt.Println(result.Extracted["source_ip"])
	fmt.Println(string(output))

}

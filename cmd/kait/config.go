package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type config struct {
	identity       identity
	hardware       string
	variant        string
	profile        string
	capabilities   []string
	o11y           string
	runMode        string
	metricsAddr    string
	buildkiteBin   string
	buildkiteToken string
	tokenFile      string
	command        string
}

func loadConfig() (config, error) {
	id, err := loadRuntimeIdentity()
	if err != nil {
		return config{}, err
	}
	cfg := config{
		identity:       id,
		hardware:       id.Hardware,
		variant:        id.Variant,
		profile:        id.Profile,
		capabilities:   id.Capabilities,
		o11y:           lowerEnv("KAIT_O11Y", "none"),
		runMode:        lowerEnv("KAIT_RUN_MODE", "agent"),
		metricsAddr:    os.Getenv("KAIT_METRICS_ADDR"),
		buildkiteBin:   envOr("BUILDKITE_AGENT_BIN", defaultBuildkiteAgentBinary()),
		buildkiteToken: os.Getenv("BUILDKITE_AGENT_TOKEN"),
		tokenFile:      os.Getenv("BUILDKITE_AGENT_TOKEN_FILE"),
		command:        os.Getenv("KAIT_COMMAND"),
	}
	if err := validateHardware(cfg.hardware); err != nil {
		return config{}, err
	}
	if err := validateVariant(cfg.variant); err != nil {
		return config{}, err
	}
	switch cfg.o11y {
	case "none", "prometheus", "datadog", "splunk":
	default:
		return config{}, fmt.Errorf("KAIT_O11Y must be one of none, prometheus, datadog, splunk (got %q)", cfg.o11y)
	}
	switch cfg.runMode {
	case "agent":
		if cfg.buildkiteToken == "" && cfg.tokenFile == "" {
			return config{}, errors.New("BUILDKITE_AGENT_TOKEN or BUILDKITE_AGENT_TOKEN_FILE is required in agent mode")
		}
		if cfg.buildkiteToken != "" && cfg.tokenFile != "" {
			return config{}, errors.New("set only one of BUILDKITE_AGENT_TOKEN or BUILDKITE_AGENT_TOKEN_FILE")
		}
		if cfg.tokenFile != "" {
			if _, err := os.Stat(cfg.tokenFile); err != nil {
				return config{}, fmt.Errorf("BUILDKITE_AGENT_TOKEN_FILE: %w", err)
			}
		}
	case "command":
		if cfg.command == "" {
			return config{}, errors.New("KAIT_COMMAND is required in command mode")
		}
	default:
		return config{}, fmt.Errorf("KAIT_RUN_MODE must be agent or command (got %q)", cfg.runMode)
	}
	if cfg.metricsAddr == "" && cfg.o11y != "none" {
		cfg.metricsAddr = ":9090"
	}
	return cfg, nil
}

func lowerEnv(name, fallback string) string {
	return strings.ToLower(envOr(name, fallback))
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y":
		return true
	default:
		return false
	}
}

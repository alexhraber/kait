package main

import (
	"os"
	"strings"
	"time"
)

func main() {
	if err := configureHardwareEnvironment(); err != nil {
		logEvent("error", "hardware_environment_failed", map[string]string{"error": err.Error()})
		os.Exit(1)
	}
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "doctor":
			if err := writeDoctor(os.Stdout); err != nil {
				logEvent("error", "doctor_failed", map[string]string{"error": err.Error()})
				os.Exit(1)
			}
			return
		case "smoke":
			if err := writeSmoke(os.Stdout); err != nil {
				logEvent("error", "smoke_failed", map[string]string{"error": err.Error()})
				os.Exit(1)
			}
			return
		case "hardware":
			if err := writeHardware(os.Stdout); err != nil {
				logEvent("error", "hardware_check_failed", map[string]string{"error": err.Error()})
				os.Exit(1)
			}
			return
		}
	}

	cfg, err := loadConfig()
	if err != nil {
		logEvent("error", "configuration_failed", map[string]string{"error": err.Error()})
		os.Exit(2)
	}

	stats := &metrics{
		startTime:    time.Now().UTC(),
		hardware:     cfg.hardware,
		variant:      cfg.variant,
		capabilities: strings.Join(cfg.capabilities, ","),
		o11y:         cfg.o11y,
	}
	logEvent("info", "runtime_starting", map[string]string{
		"hardware":     cfg.hardware,
		"variant":      cfg.variant,
		"capabilities": strings.Join(cfg.capabilities, ","),
		"o11y":         cfg.o11y,
		"run_mode":     cfg.runMode,
	})

	server, listener, err := startMetricsServer(cfg.metricsAddr, stats)
	if err != nil {
		logEvent("error", "metrics_server_failed", map[string]string{"error": err.Error()})
		os.Exit(1)
	}
	if server != nil {
		logEvent("info", "metrics_server_started", map[string]string{"address": listener.Addr().String()})
	}

	dog, err := newDogStatsD(cfg)
	if err != nil {
		logEvent("error", "datadog_setup_failed", map[string]string{"error": err.Error()})
		shutdownServer(server)
		os.Exit(1)
	}
	if dog != nil {
		defer dog.Close()
	}

	exitCode := run(ctxWithSignals(), cfg, stats, dog)
	shutdownServer(server)
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

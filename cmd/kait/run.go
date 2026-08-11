package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

func run(ctx context.Context, cfg config, stats *metrics, dog *dogStatsD) int {
	var command *exec.Cmd
	if cfg.runMode == "agent" {
		agentBin := cfg.buildkiteBin
		if _, err := os.Stat(agentBin); err != nil {
			if resolved, lookErr := exec.LookPath(agentBin); lookErr == nil {
				agentBin = resolved
			} else {
				logEvent("error", "agent_binary_missing", map[string]string{"path": cfg.buildkiteBin})
				return 1
			}
		}
		agentArgs, err := buildAgentArgs(cfg)
		if err != nil {
			logEvent("error", "agent_configuration_failed", map[string]string{"error": err.Error()})
			return 2
		}
		command = exec.Command(agentBin, agentArgs...)
		logEvent("info", "buildkite_agent_starting", map[string]string{"binary": agentBin})
	} else {
		command = exec.Command("/bin/sh", "-lc", cfg.command)
		logEvent("info", "command_starting", map[string]string{"command": cfg.command})
	}
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.Stdin = os.Stdin

	if err := command.Start(); err != nil {
		logEvent("error", "process_start_failed", map[string]string{"error": err.Error()})
		return 1
	}
	stats.starts.Add(1)
	stats.running.Store(1)
	if dog != nil {
		dog.gauge("kait.agent.running", 1)
		dog.count("kait.agent.starts", 1)
	}

	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case err := <-done:
		return finishProcess(err, stats, dog)
	case <-ctx.Done():
		logEvent("info", "shutdown_requested", map[string]string{"signal": "SIGTERM"})
		_ = command.Process.Signal(syscall.SIGTERM)
		err := <-done
		if err == nil {
			return 0
		}
		return finishProcess(err, stats, dog)
	}
}

func finishProcess(err error, stats *metrics, dog *dogStatsD) int {
	stats.running.Store(0)
	stats.exits.Add(1)
	code := 0
	if err != nil {
		code = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
			if code < 0 {
				code = 1
			}
		}
	}
	stats.lastExit.Store(int64(code))
	if dog != nil {
		dog.gauge("kait.agent.running", 0)
		dog.count("kait.agent.exits", 1)
	}
	logEvent("info", "runtime_stopped", map[string]string{"exit_code": strconv.Itoa(code)})
	return code
}

func buildAgentArgs(cfg config) ([]string, error) {
	args := []string{"start"}
	if cfg.tokenFile != "" {
		args = append(args, "--token", "file://"+cfg.tokenFile)
	}
	id := cfg.identity
	if id.Hardware == "" {
		id = identity{
			Schema:       1,
			Hardware:     cfg.hardware,
			Variant:      cfg.variant,
			Profile:      cfg.profile,
			Capabilities: cfg.capabilities,
		}
		if id.Profile == "" {
			id.Profile = profileForVariant(id.Variant)
		}
		if len(id.Capabilities) == 0 {
			id.Capabilities = capabilitiesForVariant(id.Variant)
		}
	}
	tags, err := mergeAgentTags(os.Getenv("BUILDKITE_AGENT_TAGS"), id, cfg.o11y)
	if err != nil {
		return nil, err
	}
	args = append(args, "--tags", tags)
	appendEnvFlag := func(name, flag string) {
		if value := os.Getenv(name); value != "" {
			args = append(args, flag, value)
		}
	}
	appendEnvFlag("BUILDKITE_AGENT_NAME", "--name")
	appendEnvFlag("BUILDKITE_AGENT_CONFIG", "--config")
	appendEnvFlag("BUILDKITE_AGENT_ENDPOINT", "--endpoint")
	appendEnvFlag("BUILDKITE_AGENT_QUEUE", "--queue")
	appendEnvFlag("BUILDKITE_AGENT_PRIORITY", "--priority")
	appendEnvFlag("BUILDKITE_AGENT_ACQUIRE_JOB", "--acquire-job")
	appendEnvFlag("BUILDKITE_AGENT_DISCONNECT_AFTER_IDLE_TIMEOUT", "--disconnect-after-idle-timeout")
	appendEnvFlag("BUILDKITE_AGENT_SHELL", "--shell")
	if truthy(os.Getenv("BUILDKITE_AGENT_DISCONNECT_AFTER_JOB")) {
		args = append(args, "--disconnect-after-job")
	}
	if truthy(os.Getenv("BUILDKITE_AGENT_REFLECT_EXIT_STATUS")) {
		args = append(args, "--reflect-exit-status")
	}
	if truthy(os.Getenv("BUILDKITE_WRITE_JOB_LOGS_TO_STDOUT")) {
		args = append(args, "--write-job-logs-to-stdout")
	}
	if truthy(os.Getenv("BUILDKITE_KUBERNETES_EXEC")) {
		args = append(args, "--kubernetes-exec")
	}
	return args, nil
}

// ctxWithSignals cancels on SIGINT/SIGTERM for the process lifetime.
// The cancel function is intentionally not returned; process exit is the cleanup.
func ctxWithSignals() context.Context {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	// stop is not called on purpose: the process exits after the child does.
	// Keeping the notify active for the process lifetime matches prior behavior.
	_ = stop
	return ctx
}

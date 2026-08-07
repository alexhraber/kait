package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

const version = "0.1.0"

type config struct {
	hardware       string
	variant        string
	o11y           string
	runMode        string
	metricsAddr    string
	buildkiteBin   string
	buildkiteToken string
	tokenFile      string
	command        string
}

type metrics struct {
	starts    atomic.Uint64
	exits     atomic.Uint64
	running   atomic.Int64
	lastExit  atomic.Int64
	startTime time.Time
	hardware  string
	variant   string
	o11y      string
}

type dogStatsD struct {
	conn net.Conn
	tags string
}

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
		startTime: time.Now().UTC(),
		hardware:  cfg.hardware,
		variant:   cfg.variant,
		o11y:      cfg.o11y,
	}
	logEvent("info", "runtime_starting", map[string]string{
		"hardware": cfg.hardware,
		"variant":  cfg.variant,
		"o11y":     cfg.o11y,
		"run_mode": cfg.runMode,
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

func loadConfig() (config, error) {
	cfg := config{
		hardware:       lowerEnv("KAITE_HARDWARE", "cpu"),
		variant:        lowerEnv("KAITE_VARIANT", "slim"),
		o11y:           lowerEnv("KAITE_O11Y", "none"),
		runMode:        lowerEnv("KAITE_RUN_MODE", "agent"),
		metricsAddr:    os.Getenv("KAITE_METRICS_ADDR"),
		buildkiteBin:   envOr("BUILDKITE_AGENT_BIN", "/buildkite/bin/buildkite-agent"),
		buildkiteToken: os.Getenv("BUILDKITE_AGENT_TOKEN"),
		tokenFile:      os.Getenv("BUILDKITE_AGENT_TOKEN_FILE"),
		command:        os.Getenv("KAITE_COMMAND"),
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
		return config{}, fmt.Errorf("KAITE_O11Y must be one of none, prometheus, datadog, splunk (got %q)", cfg.o11y)
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
			return config{}, errors.New("KAITE_COMMAND is required in command mode")
		}
	default:
		return config{}, fmt.Errorf("KAITE_RUN_MODE must be agent or command (got %q)", cfg.runMode)
	}
	if cfg.metricsAddr == "" && cfg.o11y != "none" {
		cfg.metricsAddr = ":9090"
	}
	return cfg, nil
}

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
		command = exec.Command(agentBin, buildAgentArgs(cfg)...)
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
		dog.gauge("kaite.agent.running", 1)
		dog.count("kaite.agent.starts", 1)
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
		dog.gauge("kaite.agent.running", 0)
		dog.count("kaite.agent.exits", 1)
	}
	logEvent("info", "runtime_stopped", map[string]string{"exit_code": strconv.Itoa(code)})
	return code
}

func buildAgentArgs(cfg config) []string {
	args := []string{"start"}
	if cfg.tokenFile != "" {
		args = append(args, "--token", "file://"+cfg.tokenFile)
	}
	if tags := os.Getenv("BUILDKITE_AGENT_TAGS"); tags != "" {
		args = append(args, "--tags", tags)
	} else {
		variant := cfg.variant
		if variant == "" {
			variant = "slim"
		}
		args = append(args, "--tags", "kaite=true,kaite.hardware="+cfg.hardware+",kaite.variant="+variant+",kaite.o11y="+cfg.o11y)
	}
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
	return args
}

func startMetricsServer(addr string, stats *metrics) (*http.Server, net.Listener, error) {
	if addr == "" {
		return nil, nil, nil
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if stats.running.Load() == 1 {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = io.WriteString(w, stats.prometheus())
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logEvent("error", "metrics_server_stopped", map[string]string{"error": err.Error()})
		}
	}()
	return server, listener, nil
}

func shutdownServer(server *http.Server) {
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}

func (m *metrics) prometheus() string {
	labels := "hardware=\"" + prometheusLabel(m.hardware) + "\",variant=\"" + prometheusLabel(m.variant) + "\",o11y=\"" + prometheusLabel(m.o11y) + "\""
	var b strings.Builder
	fmt.Fprintf(&b, "# HELP kaite_info Kaite runtime build and configuration information.\n# TYPE kaite_info gauge\nkaite_info{version=\"%s\",%s} 1\n", version, labels)
	fmt.Fprintf(&b, "# HELP kaite_agent_starts_total Number of child process starts.\n# TYPE kaite_agent_starts_total counter\nkaite_agent_starts_total{%s} %d\n", labels, m.starts.Load())
	fmt.Fprintf(&b, "# HELP kaite_agent_exits_total Number of child process exits.\n# TYPE kaite_agent_exits_total counter\nkaite_agent_exits_total{%s} %d\n", labels, m.exits.Load())
	fmt.Fprintf(&b, "# HELP kaite_agent_running Whether the child process is running.\n# TYPE kaite_agent_running gauge\nkaite_agent_running{%s} %d\n", labels, m.running.Load())
	fmt.Fprintf(&b, "# HELP kaite_agent_last_exit_code Last child process exit code.\n# TYPE kaite_agent_last_exit_code gauge\nkaite_agent_last_exit_code{%s} %d\n", labels, m.lastExit.Load())
	fmt.Fprintf(&b, "# HELP process_start_time_seconds Unix time when the Kaite runtime started.\n# TYPE process_start_time_seconds gauge\nprocess_start_time_seconds %.3f\n", float64(m.startTime.UnixNano())/1e9)
	return b.String()
}

func newDogStatsD(cfg config) (*dogStatsD, error) {
	if cfg.o11y != "datadog" {
		return nil, nil
	}
	host := envOr("KAITE_DD_AGENT_HOST", envOr("DD_AGENT_HOST", "127.0.0.1"))
	port := envOr("KAITE_DD_DOGSTATSD_PORT", envOr("DD_DOGSTATSD_PORT", "8125"))
	conn, err := net.Dial("udp", net.JoinHostPort(host, port))
	if err != nil {
		return nil, err
	}
	tags := "kaite_hardware:" + cfg.hardware + ",kaite_variant:" + cfg.variant + ",kaite_o11y:datadog"
	return &dogStatsD{conn: conn, tags: tags}, nil
}

func (d *dogStatsD) Close() {
	_ = d.conn.Close()
}

func (d *dogStatsD) gauge(name string, value int) {
	d.send(name, strconv.Itoa(value)+"|g")
}

func (d *dogStatsD) count(name string, value int) {
	d.send(name, strconv.Itoa(value)+"|c")
}

func (d *dogStatsD) send(name, value string) {
	_, _ = fmt.Fprintf(d.conn, "%s:%s|#%s\n", name, value, d.tags)
}

func writeDoctor(w io.Writer) error {
	result := map[string]any{
		"version": version,
		"variant": lowerEnv("KAITE_VARIANT", "slim"),
		"hardware": map[string]bool{
			"cpu":    true,
			"apple":  true,
			"nvidia": commandAvailable("nvidia-smi"),
			"amd":    commandAvailable("rocminfo"),
			"intel":  commandAvailable("sycl-ls"),
		},
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func writeSmoke(w io.Writer) error {
	hardware := lowerEnv("KAITE_HARDWARE", "cpu")
	variant := lowerEnv("KAITE_VARIANT", "slim")
	if err := validateVariant(variant); err != nil {
		return err
	}
	var frameworkCheck string
	switch hardware {
	case "cpu", "apple":
		if hardware == "apple" && runtime.GOARCH != "arm64" {
			return fmt.Errorf("apple hardware target requires linux/arm64 (running on %s)", runtime.GOARCH)
		}
		frameworkCheck = `import numpy, sklearn, torch; print("cpu toolchain ready")`
		if variant == "full" {
			frameworkCheck = `import accelerate, datasets, diffusers, fastapi, gradio, lightning, mlflow, ray, torch, transformers, uvicorn, wandb; print("full AI/ML toolchain ready")`
		}
	case "nvidia", "amd":
		if err := writeHardware(w); err != nil {
			return err
		}
		frameworkCheck = `import torch; assert torch.cuda.is_available(), "torch cannot see the accelerator"; print(torch.cuda.get_device_name(0))`
		if variant == "full" {
			frameworkCheck = `import accelerate, datasets, diffusers, fastapi, gradio, lightning, mlflow, ray, transformers, uvicorn, wandb; import torch; assert torch.cuda.is_available(), "torch cannot see the accelerator"; print(torch.cuda.get_device_name(0))`
		}
	case "intel":
		if err := writeHardware(w); err != nil {
			return err
		}
		frameworkCheck = `import torch, intel_extension_for_pytorch; assert torch.xpu.is_available(), "torch cannot see the XPU"; print(torch.xpu.get_device_name(0))`
		if variant == "full" {
			frameworkCheck = `import accelerate, datasets, diffusers, fastapi, gradio, lightning, mlflow, ray, transformers, uvicorn, wandb; import torch, intel_extension_for_pytorch; assert torch.xpu.is_available(), "torch cannot see the XPU"; print(torch.xpu.get_device_name(0))`
		}
	default:
		return fmt.Errorf("unsupported KAITE_HARDWARE=%s", hardware)
	}
	if err := runCheck(w, "python", "-c", frameworkCheck); err != nil {
		return fmt.Errorf("%s framework check: %w", hardware, err)
	}
	fmt.Fprintf(w, "kaite smoke: %s-%s ready\n", hardware, variant)
	return nil
}

func writeHardware(w io.Writer) error {
	hardware := lowerEnv("KAITE_HARDWARE", "cpu")
	switch hardware {
	case "cpu":
		fmt.Fprintln(w, "cpu")
		return nil
	case "apple":
		if runtime.GOARCH != "arm64" {
			return fmt.Errorf("apple hardware target requires linux/arm64 (running on %s)", runtime.GOARCH)
		}
		fmt.Fprintln(w, "apple (linux/arm64 CPU; Apple GPU is not exposed by Ubuntu containers)")
		return nil
	case "nvidia":
		return runCheck(w, "nvidia-smi", "--query-gpu=name,driver_version,memory.total", "--format=csv,noheader")
	case "amd":
		return runCheck(w, "rocminfo")
	case "intel":
		return runCheck(w, "sycl-ls")
	default:
		return fmt.Errorf("unsupported KAITE_HARDWARE=%s", hardware)
	}
}

func validateHardware(hardware string) error {
	switch hardware {
	case "cpu", "apple", "nvidia", "amd", "intel":
		return nil
	default:
		return fmt.Errorf("KAITE_HARDWARE must be one of cpu, apple, nvidia, amd, intel (got %q)", hardware)
	}
}

func validateVariant(variant string) error {
	switch variant {
	case "slim", "full":
		return nil
	default:
		return fmt.Errorf("KAITE_VARIANT must be one of slim, full (got %q)", variant)
	}
}

func configureHardwareEnvironment() error {
	if lowerEnv("KAITE_HARDWARE", "cpu") != "intel" {
		return nil
	}
	setvars := "/opt/intel/oneapi/setvars.sh"
	if _, err := os.Stat(setvars); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	command := exec.Command("bash", "-c", "source /opt/intel/oneapi/setvars.sh >/dev/null 2>&1 && env -0")
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("source %s: %w", setvars, err)
	}
	for _, entry := range bytes.Split(output, []byte{0}) {
		separator := bytes.IndexByte(entry, '=')
		if separator <= 0 {
			continue
		}
		if err := os.Setenv(string(entry[:separator]), string(entry[separator+1:])); err != nil {
			return fmt.Errorf("set oneAPI environment: %w", err)
		}
	}
	return nil
}

func runCheck(w io.Writer, name string, args ...string) error {
	command := exec.Command(name, args...)
	output, err := command.CombinedOutput()
	if len(output) > 0 {
		_, _ = w.Write(output)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func ctxWithSignals() context.Context {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	return ctx
}

func logEvent(level, event string, fields map[string]string) {
	entry := map[string]any{
		"ts":        time.Now().UTC().Format(time.RFC3339Nano),
		"level":     level,
		"component": "kaite",
		"event":     event,
	}
	for key, value := range fields {
		entry[key] = value
	}
	data, _ := json.Marshal(entry)
	_, _ = fmt.Fprintln(os.Stderr, string(data))
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

func prometheusLabel(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return strings.ReplaceAll(value, "\n", "\\n")
}

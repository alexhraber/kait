package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigCommandMode(t *testing.T) {
	t.Setenv("KAITE_HARDWARE", "nvidia")
	t.Setenv("KAITE_VARIANT", "full")
	t.Setenv("KAITE_O11Y", "prometheus")
	t.Setenv("KAITE_RUN_MODE", "command")
	t.Setenv("KAITE_COMMAND", "echo ready")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.metricsAddr != ":9090" {
		t.Fatalf("metrics address = %q, want :9090", cfg.metricsAddr)
	}
}

func TestLoadConfigRejectsUnknownO11y(t *testing.T) {
	t.Setenv("KAITE_RUN_MODE", "command")
	t.Setenv("KAITE_COMMAND", "true")
	t.Setenv("KAITE_O11Y", "honeycomb")
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "KAITE_O11Y") {
		t.Fatalf("loadConfig() error = %v, want KAITE_O11Y validation", err)
	}
}

func TestLoadConfigRejectsUnknownHardware(t *testing.T) {
	t.Setenv("KAITE_RUN_MODE", "command")
	t.Setenv("KAITE_COMMAND", "true")
	t.Setenv("KAITE_HARDWARE", "armv7")
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "KAITE_HARDWARE") {
		t.Fatalf("loadConfig() error = %v, want KAITE_HARDWARE validation", err)
	}
}

func TestLoadConfigRejectsUnknownVariant(t *testing.T) {
	t.Setenv("KAITE_RUN_MODE", "command")
	t.Setenv("KAITE_COMMAND", "true")
	t.Setenv("KAITE_VARIANT", "max")
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "KAITE_VARIANT") {
		t.Fatalf("loadConfig() error = %v, want KAITE_VARIANT validation", err)
	}
}

func TestHardwareTargetsIncludeAppleSilicon(t *testing.T) {
	for _, hardware := range []string{"cpu", "apple", "nvidia", "amd", "intel"} {
		if err := validateHardware(hardware); err != nil {
			t.Fatalf("validateHardware(%q) error = %v", hardware, err)
		}
	}
}

func TestBuildAgentArgsUsesFileTokenAndHardwareTags(t *testing.T) {
	t.Setenv("BUILDKITE_AGENT_TAGS", "queue=ai")
	t.Setenv("BUILDKITE_AGENT_QUEUE", "ai")
	t.Setenv("BUILDKITE_AGENT_ENDPOINT", "https://agent.example.test/v3")
	cfg := config{hardware: "amd", variant: "full", o11y: "splunk", tokenFile: "/run/secrets/buildkite-agent-token"}
	argsList, err := buildAgentArgs(cfg)
	if err != nil {
		t.Fatalf("buildAgentArgs() error = %v", err)
	}
	args := strings.Join(argsList, " ")
	if !strings.Contains(args, "--token file:///run/secrets/buildkite-agent-token") {
		t.Fatalf("args = %q, missing file token", args)
	}
	if !strings.Contains(args, "--tags queue=ai") {
		t.Fatalf("args = %q, missing explicit tags", args)
	}
	if !strings.Contains(args, "--queue ai") || !strings.Contains(args, "--endpoint https://agent.example.test/v3") {
		t.Fatalf("args = %q, missing standard agent routing options", args)
	}

	secretArgsList, err := buildAgentArgs(config{hardware: "cpu", o11y: "none", buildkiteToken: "super-secret"})
	if err != nil {
		t.Fatalf("buildAgentArgs() error = %v", err)
	}
	secretArgs := strings.Join(secretArgsList, " ")
	if strings.Contains(secretArgs, "super-secret") {
		t.Fatalf("args = %q, raw token must remain in the inherited environment", secretArgs)
	}
}

func TestLoadConfigRejectsTwoTokenSources(t *testing.T) {
	t.Setenv("KAITE_RUN_MODE", "agent")
	t.Setenv("BUILDKITE_AGENT_TOKEN", "env-token")
	t.Setenv("BUILDKITE_AGENT_TOKEN_FILE", "/dev/null")
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "only one") {
		t.Fatalf("loadConfig() error = %v, want mutually exclusive token sources", err)
	}
}

func TestPrometheusMetricsExposeRuntimeLabels(t *testing.T) {
	m := &metrics{hardware: "intel", variant: "full", capabilities: "data-science,training", o11y: "prometheus"}
	m.starts.Store(2)
	m.running.Store(1)
	output := m.prometheus()
	for _, want := range []string{
		"kaite_info{version=\"0.2.1\",hardware=\"intel\",variant=\"full\",capabilities=\"data-science,training\",o11y=\"prometheus\"} 1",
		"kaite_agent_starts_total{hardware=\"intel\",variant=\"full\",capabilities=\"data-science,training\",o11y=\"prometheus\"} 2",
		"kaite_agent_running{hardware=\"intel\",variant=\"full\",capabilities=\"data-science,training\",o11y=\"prometheus\"} 1",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("metrics missing %q in %s", want, output)
		}
	}
}

func TestBuildAgentArgsAddsCapabilityTagsAndRejectsOverrides(t *testing.T) {
	t.Setenv("BUILDKITE_AGENT_TAGS", "queue=ai,custom=true")
	cfg := config{
		identity: identity{
			Schema:       1,
			Hardware:     "cpu",
			Variant:      "full",
			Capabilities: []string{"data-science", "training", "orchestration", "serving"},
		},
		o11y: "none",
	}
	args, err := buildAgentArgs(cfg)
	if err != nil {
		t.Fatalf("buildAgentArgs() error = %v", err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{
		"queue=ai",
		"custom=true",
		"kaite=true",
		"kaite.hardware=cpu",
		"kaite.capability.training=true",
		"kaite.capability.serving=true",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args = %q, missing %q", joined, want)
		}
	}

	t.Setenv("BUILDKITE_AGENT_TAGS", "kaite.hardware=nvidia")
	if _, err := buildAgentArgs(cfg); err == nil || !strings.Contains(err.Error(), "reserved tag") {
		t.Fatalf("buildAgentArgs() error = %v, want reserved-tag rejection", err)
	}
}

func TestRuntimeIdentityRejectsConflictingEnvironment(t *testing.T) {
	oldPath := identityPath
	t.Cleanup(func() { identityPath = oldPath })
	identityPath = filepath.Join(t.TempDir(), "identity.json")
	if err := os.WriteFile(identityPath, []byte(`{"schema":1,"hardware":"cpu","variant":"slim","capabilities":["data-science"]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("KAITE_HARDWARE", "nvidia")
	if _, err := loadRuntimeIdentity(); err == nil || !strings.Contains(err.Error(), "conflicts with baked image hardware") {
		t.Fatalf("loadRuntimeIdentity() error = %v, want baked hardware conflict", err)
	}
}

func TestSmokeFrameworkCheckUsesCapabilityContract(t *testing.T) {
	check, err := smokeFrameworkCheck("cpu", []string{"data-science", "training", "serving"})
	if err != nil {
		t.Fatalf("smokeFrameworkCheck() error = %v", err)
	}
	for _, want := range []string{"numpy", "accelerate", "transformers", "fastapi", "uvicorn"} {
		if !strings.Contains(check, want) {
			t.Fatalf("check = %q, missing %q", check, want)
		}
	}
}

func TestSmokeRejectsUnknownHardware(t *testing.T) {
	t.Setenv("KAITE_HARDWARE", "quantum")
	if err := writeSmoke(&strings.Builder{}); err == nil || !strings.Contains(err.Error(), "unsupported KAITE_HARDWARE") {
		t.Fatalf("writeSmoke() error = %v, want hardware validation", err)
	}
}

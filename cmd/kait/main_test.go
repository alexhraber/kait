package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func installTestIdentity(t *testing.T, hardware, profile string) identity {
	t.Helper()
	oldPath := identityPath
	t.Cleanup(func() { identityPath = oldPath })
	identity, err := resolveContract(hardware, profile, "", "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "identity.json")
	data, err := json.Marshal(identity)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	identityPath = path
	for _, name := range []string{"KAIT_HARDWARE", "KAIT_VARIANT", "KAIT_PROFILE", "KAIT_CAPABILITIES"} {
		t.Setenv(name, "")
	}
	return identity
}

func TestCapabilityContractDefinesOfficialProfilesAndHardware(t *testing.T) {
	wantHardware := []string{"cpu", "apple", "nvidia", "amd", "intel"}
	if strings.Join(supportedHardwareNames(), ",") != strings.Join(wantHardware, ",") {
		t.Fatalf("hardware = %v, want %v", supportedHardwareNames(), wantHardware)
	}
	wantProfiles := []string{"slim", "full", "data-science", "training", "orchestration", "serving"}
	if strings.Join(profileNames(), ",") != strings.Join(wantProfiles, ",") {
		t.Fatalf("profiles = %v, want %v", profileNames(), wantProfiles)
	}
	for _, hardware := range wantHardware {
		for _, profile := range wantProfiles {
			identity, err := resolveContract(hardware, profile, "", "")
			if err != nil {
				t.Fatalf("resolveContract(%s,%s): %v", hardware, profile, err)
			}
			if err := validateIdentity(identity); err != nil {
				t.Fatalf("validateIdentity(%s,%s): %v", hardware, profile, err)
			}
		}
	}
}

func TestCapabilityCompositionIsIntentional(t *testing.T) {
	if got := capabilitiesForProfile("slim"); !sameCapabilities(got, []string{"data-science"}) {
		t.Fatalf("slim capabilities = %v", got)
	}
	if got := capabilitiesForProfile("training"); !sameCapabilities(got, []string{"data-science", "training"}) {
		t.Fatalf("training capabilities = %v", got)
	}
	if got := capabilitiesForProfile("orchestration"); !sameCapabilities(got, []string{"orchestration"}) {
		t.Fatalf("orchestration capabilities = %v", got)
	}
	if got := capabilitiesForProfile("serving"); !sameCapabilities(got, []string{"serving"}) {
		t.Fatalf("serving capabilities = %v", got)
	}
	if got := capabilitiesForProfile("full"); !sameCapabilities(got, []string{"data-science", "training", "orchestration", "serving"}) {
		t.Fatalf("full capabilities = %v", got)
	}
}

func TestLoadConfigCommandModeUsesBakedIdentity(t *testing.T) {
	id := installTestIdentity(t, "nvidia", "full")
	t.Setenv("KAIT_RUN_MODE", "command")
	t.Setenv("KAIT_COMMAND", "echo ready")
	t.Setenv("KAIT_O11Y", "prometheus")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if cfg.profile != id.Profile || cfg.metricsAddr != ":9090" {
		t.Fatalf("config = %+v, want profile %q and metrics :9090", cfg, id.Profile)
	}
}

func TestLoadConfigRejectsRuntimeIdentityOverrides(t *testing.T) {
	installTestIdentity(t, "cpu", "slim")
	t.Setenv("KAIT_RUN_MODE", "command")
	t.Setenv("KAIT_COMMAND", "true")
	t.Setenv("KAIT_HARDWARE", "nvidia")
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "conflicts with baked image hardware") {
		t.Fatalf("loadConfig() error = %v, want baked hardware conflict", err)
	}
}

func TestLoadConfigRejectsUnknownO11y(t *testing.T) {
	installTestIdentity(t, "cpu", "slim")
	t.Setenv("KAIT_RUN_MODE", "command")
	t.Setenv("KAIT_COMMAND", "true")
	t.Setenv("KAIT_O11Y", "honeycomb")
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "KAIT_O11Y") {
		t.Fatalf("loadConfig() error = %v, want KAIT_O11Y validation", err)
	}
}

func TestLoadConfigRejectsTwoTokenSources(t *testing.T) {
	installTestIdentity(t, "cpu", "slim")
	t.Setenv("KAIT_RUN_MODE", "agent")
	t.Setenv("BUILDKITE_AGENT_TOKEN", "env-token")
	t.Setenv("BUILDKITE_AGENT_TOKEN_FILE", "/dev/null")
	if _, err := loadConfig(); err == nil || !strings.Contains(err.Error(), "only one") {
		t.Fatalf("loadConfig() error = %v, want mutually exclusive token sources", err)
	}
}

func TestBuildAgentArgsUsesIdentityTagsAndRejectsOverrides(t *testing.T) {
	t.Setenv("BUILDKITE_AGENT_TAGS", "queue=ai,custom=true")
	identity := installTestIdentity(t, "cpu", "full")
	argsList, err := buildAgentArgs(config{identity: identity, hardware: identity.Hardware, variant: identity.Variant, profile: identity.Profile, capabilities: identity.Capabilities, o11y: "none"})
	if err != nil {
		t.Fatalf("buildAgentArgs() error = %v", err)
	}
	joined := strings.Join(argsList, " ")
	for _, want := range []string{
		"queue=ai",
		"custom=true",
		"kait=true",
		"kait.hardware=cpu",
		"kait.profile=full",
		"kait.capability.training=true",
		"kait.capability.serving=true",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args = %q, missing %q", joined, want)
		}
	}

	t.Setenv("BUILDKITE_AGENT_TAGS", "kait.hardware=nvidia")
	if _, err := buildAgentArgs(config{identity: identity, hardware: identity.Hardware, variant: identity.Variant, profile: identity.Profile, capabilities: identity.Capabilities, o11y: "none"}); err == nil || !strings.Contains(err.Error(), "reserved tag") {
		t.Fatalf("buildAgentArgs() error = %v, want reserved-tag rejection", err)
	}
}

func TestRuntimeIdentityRequiresBakedFile(t *testing.T) {
	oldPath := identityPath
	t.Cleanup(func() { identityPath = oldPath })
	identityPath = filepath.Join(t.TempDir(), "missing.json")
	if _, err := loadRuntimeIdentity(); err == nil || !strings.Contains(err.Error(), "baked image identity is missing") {
		t.Fatalf("loadRuntimeIdentity() error = %v, want missing baked identity", err)
	}
}

func TestSmokeFrameworkCheckCoversRepresentativeContracts(t *testing.T) {
	check, err := smokeFrameworkCheck("cpu", []string{"data-science", "training", "orchestration", "serving"})
	if err != nil {
		t.Fatalf("smokeFrameworkCheck() error = %v", err)
	}
	for _, want := range []string{"numpy", "pandas", "jupyterlab", "torch", "Dataset", "TrainingArguments", "lightning", "ray", "mlflow", "wandb", "FastAPI", "gradio", "uvicorn"} {
		if !strings.Contains(check, want) {
			t.Fatalf("check is missing %q", want)
		}
	}
	if strings.Contains(check, `\nassert`) {
		t.Fatalf("check contains a literal newline escape: %q", check)
	}
}

func TestMatrixIsDerivedFromContract(t *testing.T) {
	var output bytes.Buffer
	if err := writeMatrix(&output, []string{"--active-only"}); err != nil {
		t.Fatal(err)
	}
	var matrix struct {
		Include []matrixEntry `json:"include"`
	}
	if err := json.Unmarshal(output.Bytes(), &matrix); err != nil {
		t.Fatal(err)
	}
	if len(matrix.Include) != 12 {
		t.Fatalf("active matrix entries = %d, want 12", len(matrix.Include))
	}
	for _, entry := range matrix.Include {
		if entry.Hardware != "cpu" && entry.Hardware != "apple" {
			t.Fatalf("inactive hardware in active matrix: %+v", entry)
		}
		if entry.Capabilities == "" || entry.Target == "" {
			t.Fatalf("incomplete matrix entry: %+v", entry)
		}
	}
}

func TestDoctorReportsIdentityAndHardwareContract(t *testing.T) {
	installTestIdentity(t, "cpu", "serving")
	var output bytes.Buffer
	if err := writeDoctor(&output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"identity_consistent": true`) || !strings.Contains(output.String(), `"profile": "serving"`) || !strings.Contains(output.String(), `"serving": "FastAPI`) {
		t.Fatalf("doctor output missing contract fields: %s", output.String())
	}
}

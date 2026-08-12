package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type hardwareReport struct {
	Expected  string          `json:"expected"`
	Detected  []string        `json:"detected"`
	Satisfied bool            `json:"satisfied"`
	Evidence  map[string]bool `json:"evidence"`
}

func writeDoctor(w io.Writer) error {
	id, err := loadRuntimeIdentity()
	if err != nil {
		return err
	}
	report := detectHardware(id.Hardware)
	checks := make(map[string]string, len(id.Capabilities))
	for _, capability := range id.Capabilities {
		definition := authoritativeCapabilities.Capabilities[capability]
		checks[capability] = definition.Summary + "; run kait smoke"
	}
	result := map[string]any{
		"version":             version,
		"identity":            id,
		"identity_consistent": true,
		"capabilities":        id.Capabilities,
		"variant":             id.Variant,
		"profile":             id.Profile,
		"hardware": map[string]bool{
			"cpu":    report.Evidence["cpu"],
			"apple":  report.Evidence["apple"],
			"nvidia": report.Evidence["nvidia"],
			"amd":    report.Evidence["amd"],
			"intel":  report.Evidence["intel"],
		},
		"hardware_contract": report,
		"capability_checks": checks,
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func detectHardware(expected string) hardwareReport {
	evidence := map[string]bool{
		"cpu":    true,
		"apple":  detectAppleGPU(),
		"nvidia": commandAvailable("nvidia-smi"),
		"amd":    commandAvailable("rocminfo"),
		"intel":  commandAvailable("sycl-ls"),
	}
	detected := []string{"cpu"}
	for _, hardware := range supportedHardwareNames() {
		if hardware != "cpu" && evidence[hardware] {
			detected = append(detected, hardware)
		}
	}
	return hardwareReport{
		Expected:  expected,
		Detected:  detected,
		Satisfied: evidence[expected],
		Evidence:  evidence,
	}
}

func writeSmoke(w io.Writer) error {
	id, err := loadRuntimeIdentity()
	if err != nil {
		return err
	}
	if id.Hardware == "apple" || id.Hardware == "nvidia" || id.Hardware == "amd" || id.Hardware == "intel" {
		if err := writeHardware(w); err != nil {
			return err
		}
	}
	frameworkCheck, err := smokeFrameworkCheck(id.Hardware, id.Capabilities)
	if err != nil {
		return err
	}
	if err := runCheck(w, "python", "-c", frameworkCheck); err != nil {
		return fmt.Errorf("%s framework check: %w", id.Profile, err)
	}
	fmt.Fprintf(w, "kait smoke: %s-%s profile=%s capabilities=%s ready\n", id.Hardware, id.Variant, id.Profile, strings.Join(id.Capabilities, ","))
	return nil
}

func smokeFrameworkCheck(hardware string, capabilities []string) (string, error) {
	if err := validateHardware(hardware); err != nil {
		return "", err
	}
	if hardware == "apple" {
		if err := requireApplePlatform(); err != nil {
			return "", err
		}
	}
	check, err := smokeScripts(capabilities)
	if err != nil {
		return "", err
	}
	if containsCapability(capabilities, "data-science") {
		switch hardware {
		case "nvidia", "amd":
			check += "\nassert torch.cuda.is_available(), \"torch cannot see the accelerator\"\nprint(torch.cuda.get_device_name(0))"
		case "intel":
			check += "\nimport intel_extension_for_pytorch\nassert torch.xpu.is_available(), \"torch cannot see the XPU\"\nprint(torch.xpu.get_device_name(0))"
		default:
			check += "\nassert not torch.cuda.is_available(), \"CPU/Apple contract unexpectedly exposes CUDA\""
		}
	}
	if hardware == "apple" && containsCapability(capabilities, "data-science") {
		check += "\nassert torch.backends.mps.is_built(), \"PyTorch was not built with MPS support\"\nassert torch.backends.mps.is_available(), \"PyTorch cannot see the Apple GPU\"\n_mps = torch.ones((2, 2), device=\"mps\")\nassert _mps.device.type == \"mps\""
	}
	check += "\nprint(\"Kait capability contract ready\")"
	return check, nil
}

func containsCapability(capabilities []string, wanted string) bool {
	for _, capability := range capabilities {
		if capability == wanted {
			return true
		}
	}
	return false
}

func writeHardware(w io.Writer) error {
	id, err := loadRuntimeIdentity()
	if err != nil {
		return err
	}
	switch id.Hardware {
	case "cpu":
		fmt.Fprintln(w, "cpu")
		return nil
	case "apple":
		if err := requireApplePlatform(); err != nil {
			return err
		}
		if !detectAppleGPU() {
			return errors.New("Apple GPU/Metal was not detected by system_profiler")
		}
		fmt.Fprintln(w, "apple (native macOS/arm64; Metal/MPS GPU detected)")
		return nil
	case "nvidia":
		return runCheck(w, "nvidia-smi", "--query-gpu=name,driver_version,memory.total", "--format=csv,noheader")
	case "amd":
		return runCheck(w, "rocminfo")
	case "intel":
		return runCheck(w, "sycl-ls")
	default:
		return fmt.Errorf("unsupported KAIT_HARDWARE=%s", id.Hardware)
	}
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

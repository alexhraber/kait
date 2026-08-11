package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func writeDoctor(w io.Writer) error {
	id, err := loadRuntimeIdentity()
	if err != nil {
		return err
	}
	result := map[string]any{
		"version":      version,
		"identity":     id,
		"capabilities": id.Capabilities,
		"variant":      id.Variant,
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
	configuredHardware := lowerEnv("KAIT_HARDWARE", "cpu")
	if err := validateHardware(configuredHardware); err != nil {
		return fmt.Errorf("unsupported KAIT_HARDWARE=%s", configuredHardware)
	}
	id, err := loadRuntimeIdentity()
	if err != nil {
		return err
	}
	frameworkCheck, err := smokeFrameworkCheck(id.Hardware, id.Capabilities)
	if err != nil {
		return err
	}
	if id.Hardware == "nvidia" || id.Hardware == "amd" || id.Hardware == "intel" {
		if err := writeHardware(w); err != nil {
			return err
		}
	}
	if err := runCheck(w, "python", "-c", frameworkCheck); err != nil {
		return fmt.Errorf("%s framework check: %w", id.Hardware, err)
	}
	fmt.Fprintf(w, "kait smoke: %s-%s capabilities=%s ready\n", id.Hardware, id.Variant, strings.Join(id.Capabilities, ","))
	return nil
}

func smokeFrameworkCheck(hardware string, capabilities []string) (string, error) {
	if err := validateHardware(hardware); err != nil {
		return "", err
	}
	if hardware == "apple" {
		if err := requireAppleArch(); err != nil {
			return "", err
		}
	}
	imports := make([]string, 0, len(capabilities))
	messages := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		switch capability {
		case "data-science":
			imports = append(imports, "import numpy, sklearn, torch")
			messages = append(messages, "data-science")
		case "training":
			imports = append(imports, "import accelerate, datasets, diffusers, lightning, transformers")
			messages = append(messages, "training")
		case "orchestration":
			imports = append(imports, "import mlflow, ray, wandb")
			messages = append(messages, "orchestration")
		case "serving":
			imports = append(imports, "import fastapi, gradio, uvicorn")
			messages = append(messages, "serving")
		default:
			return "", fmt.Errorf("unsupported Kait capability %s", capability)
		}
	}
	check := strings.Join(imports, "; ")
	switch hardware {
	case "nvidia", "amd":
		check += `; assert torch.cuda.is_available(), "torch cannot see the accelerator"; print(torch.cuda.get_device_name(0))`
	case "intel":
		check += `; import intel_extension_for_pytorch; assert torch.xpu.is_available(), "torch cannot see the XPU"; print(torch.xpu.get_device_name(0))`
	}
	check += `; print("Kait capabilities ready: ` + strings.Join(messages, ",") + `")`
	return check, nil
}

func writeHardware(w io.Writer) error {
	id, err := loadRuntimeIdentity()
	if err != nil {
		return err
	}
	hardware := id.Hardware
	switch hardware {
	case "cpu":
		fmt.Fprintln(w, "cpu")
		return nil
	case "apple":
		if err := requireAppleArch(); err != nil {
			return err
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
		return fmt.Errorf("unsupported KAIT_HARDWARE=%s", hardware)
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

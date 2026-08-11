package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

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
	frameworkCheck, err := smokeFrameworkCheck(hardware, variant)
	if err != nil {
		return err
	}
	if hardware == "nvidia" || hardware == "amd" || hardware == "intel" {
		if err := writeHardware(w); err != nil {
			return err
		}
	}
	if err := runCheck(w, "python", "-c", frameworkCheck); err != nil {
		return fmt.Errorf("%s framework check: %w", hardware, err)
	}
	fmt.Fprintf(w, "kaite smoke: %s-%s ready\n", hardware, variant)
	return nil
}

func smokeFrameworkCheck(hardware, variant string) (string, error) {
	switch hardware {
	case "cpu", "apple":
		if hardware == "apple" {
			if err := requireAppleArch(); err != nil {
				return "", err
			}
		}
		if variant == "full" {
			return `import accelerate, datasets, diffusers, fastapi, gradio, lightning, mlflow, ray, torch, transformers, uvicorn, wandb; print("full AI/ML toolchain ready")`, nil
		}
		return `import numpy, sklearn, torch; print("cpu toolchain ready")`, nil
	case "nvidia", "amd":
		if variant == "full" {
			return `import accelerate, datasets, diffusers, fastapi, gradio, lightning, mlflow, ray, transformers, uvicorn, wandb; import torch; assert torch.cuda.is_available(), "torch cannot see the accelerator"; print(torch.cuda.get_device_name(0))`, nil
		}
		return `import torch; assert torch.cuda.is_available(), "torch cannot see the accelerator"; print(torch.cuda.get_device_name(0))`, nil
	case "intel":
		if variant == "full" {
			return `import accelerate, datasets, diffusers, fastapi, gradio, lightning, mlflow, ray, transformers, uvicorn, wandb; import torch, intel_extension_for_pytorch; assert torch.xpu.is_available(), "torch cannot see the XPU"; print(torch.xpu.get_device_name(0))`, nil
		}
		return `import torch, intel_extension_for_pytorch; assert torch.xpu.is_available(), "torch cannot see the XPU"; print(torch.xpu.get_device_name(0))`, nil
	default:
		return "", fmt.Errorf("unsupported KAITE_HARDWARE=%s", hardware)
	}
}

func writeHardware(w io.Writer) error {
	hardware := lowerEnv("KAITE_HARDWARE", "cpu")
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
		return fmt.Errorf("unsupported KAITE_HARDWARE=%s", hardware)
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

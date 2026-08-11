package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// configureHardwareEnvironment applies vendor-specific shell environments when
// present (today: Intel oneAPI setvars). Missing setvars is a no-op so the
// same binary works on CPU-only hosts.
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

func requireAppleArch() error {
	if runtime.GOARCH != "arm64" {
		return fmt.Errorf("apple hardware target requires linux/arm64 (running on %s)", runtime.GOARCH)
	}
	return nil
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

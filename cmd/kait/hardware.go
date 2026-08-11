package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// configureHardwareEnvironment applies vendor-specific shell environments when
// present (today: Intel oneAPI setvars). Missing setvars is a no-op so the
// same binary works on CPU-only hosts.
func configureHardwareEnvironment() error {
	id, err := loadRuntimeIdentity()
	if err != nil {
		return err
	}
	if id.Hardware != "intel" {
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

func requireApplePlatform() error {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return fmt.Errorf("apple hardware target requires native macOS/arm64 (running on %s/%s)", runtime.GOOS, runtime.GOARCH)
	}
	return nil
}

func detectAppleGPU() bool {
	if requireApplePlatform() != nil {
		return false
	}
	output, err := exec.Command("system_profiler", "SPDisplaysDataType", "-json").Output()
	if err != nil {
		return false
	}
	text := string(output)
	return strings.Contains(text, "Apple") && strings.Contains(text, "spdisplays")
}

func commandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

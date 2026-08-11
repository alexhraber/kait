package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const defaultIdentityPath = "/etc/kait/identity.json"

// identityPath is a variable so unit tests can exercise image/runtime
// identity checks without writing to the host filesystem. The image always
// uses the fixed default path; it is not a runtime configuration knob.
var identityPath = defaultIdentityPath

type identity struct {
	Schema       int      `json:"schema"`
	Hardware     string   `json:"hardware"`
	Variant      string   `json:"variant"`
	Capabilities []string `json:"capabilities"`
}

var supportedCapabilities = map[string]bool{
	"data-science":  true,
	"training":      true,
	"orchestration": true,
	"serving":       true,
}

func loadRuntimeIdentity() (identity, error) {
	data, err := os.ReadFile(identityPath)
	if errors.Is(err, os.ErrNotExist) {
		id := identity{
			Schema:   1,
			Hardware: lowerEnv("KAIT_HARDWARE", "cpu"),
			Variant:  lowerEnv("KAIT_VARIANT", "slim"),
		}
		capabilities := strings.TrimSpace(os.Getenv("KAIT_CAPABILITIES"))
		if capabilities == "" {
			id.Capabilities = capabilitiesForVariant(id.Variant)
		} else {
			id.Capabilities, err = parseCapabilities(capabilities)
			if err != nil {
				return identity{}, err
			}
		}
		if err := validateIdentity(id); err != nil {
			return identity{}, err
		}
		return id, nil
	}
	if err != nil {
		return identity{}, fmt.Errorf("read Kait image identity %s: %w", identityPath, err)
	}

	var id identity
	if err := json.Unmarshal(data, &id); err != nil {
		return identity{}, fmt.Errorf("parse Kait image identity %s: %w", identityPath, err)
	}
	id.Hardware = strings.ToLower(strings.TrimSpace(id.Hardware))
	id.Variant = strings.ToLower(strings.TrimSpace(id.Variant))
	if err := validateIdentity(id); err != nil {
		return identity{}, err
	}

	if value := strings.TrimSpace(os.Getenv("KAIT_HARDWARE")); value != "" && strings.ToLower(value) != id.Hardware {
		return identity{}, fmt.Errorf("KAIT_HARDWARE=%q conflicts with baked image hardware %q", value, id.Hardware)
	}
	if value := strings.TrimSpace(os.Getenv("KAIT_VARIANT")); value != "" && strings.ToLower(value) != id.Variant {
		return identity{}, fmt.Errorf("KAIT_VARIANT=%q conflicts with baked image variant %q", value, id.Variant)
	}
	if value := strings.TrimSpace(os.Getenv("KAIT_CAPABILITIES")); value != "" {
		capabilities, err := parseCapabilities(value)
		if err != nil {
			return identity{}, err
		}
		if !sameCapabilities(capabilities, id.Capabilities) {
			return identity{}, fmt.Errorf("KAIT_CAPABILITIES=%q conflicts with baked image capabilities %q", value, strings.Join(id.Capabilities, ","))
		}
	}
	return id, nil
}

func sameCapabilities(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := make(map[string]bool, len(left))
	for _, capability := range left {
		seen[capability] = true
	}
	for _, capability := range right {
		if !seen[capability] {
			return false
		}
	}
	return true
}

func validateIdentity(id identity) error {
	if id.Schema != 1 {
		return fmt.Errorf("unsupported Kait image identity schema %d", id.Schema)
	}
	if err := validateHardware(id.Hardware); err != nil {
		return err
	}
	if err := validateVariant(id.Variant); err != nil {
		return err
	}
	if len(id.Capabilities) == 0 {
		return fmt.Errorf("Kait image identity must declare at least one capability")
	}
	seen := make(map[string]bool, len(id.Capabilities))
	for _, capability := range id.Capabilities {
		if !supportedCapabilities[capability] {
			return fmt.Errorf("unsupported Kait capability %q", capability)
		}
		if seen[capability] {
			return fmt.Errorf("duplicate Kait capability %q", capability)
		}
		seen[capability] = true
	}
	if !seen["data-science"] {
		return fmt.Errorf("Kait image identity must include data-science capability")
	}
	return nil
}

func capabilitiesForVariant(variant string) []string {
	if variant == "full" {
		return []string{"data-science", "training", "orchestration", "serving"}
	}
	return []string{"data-science"}
}

func parseCapabilities(value string) ([]string, error) {
	var capabilities []string
	seen := make(map[string]bool)
	for _, raw := range strings.Split(value, ",") {
		capability := strings.ToLower(strings.TrimSpace(raw))
		if capability == "" {
			return nil, fmt.Errorf("KAIT_CAPABILITIES contains an empty capability")
		}
		if !supportedCapabilities[capability] {
			return nil, fmt.Errorf("unsupported Kait capability %q", capability)
		}
		if seen[capability] {
			return nil, fmt.Errorf("duplicate Kait capability %q", capability)
		}
		seen[capability] = true
		capabilities = append(capabilities, capability)
	}
	if len(capabilities) == 0 {
		return nil, fmt.Errorf("KAIT_CAPABILITIES must not be empty")
	}
	for _, capability := range capabilities {
		if capability == "data-science" {
			return capabilities, nil
		}
	}
	return nil, fmt.Errorf("KAIT_CAPABILITIES must include data-science")
}

func canonicalAgentTags(id identity, o11y string) []string {
	tags := []string{
		"kait=true",
		"kait.hardware=" + id.Hardware,
		"kait.variant=" + id.Variant,
		"kait.o11y=" + o11y,
	}
	for _, capability := range id.Capabilities {
		tags = append(tags, "kait.capability."+capability+"=true")
	}
	return tags
}

func mergeAgentTags(custom string, id identity, o11y string) (string, error) {
	canonical := canonicalAgentTags(id, o11y)
	expected := make(map[string]string, len(canonical))
	for _, tag := range canonical {
		key, value := splitAgentTag(tag)
		expected[key] = value
	}

	merged := make([]string, 0, len(canonical)+1)
	for _, raw := range strings.Split(custom, ",") {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		key, value := splitAgentTag(tag)
		if key == "kait" || strings.HasPrefix(key, "kait.") {
			want, ok := expected[key]
			if !ok || value != want {
				return "", fmt.Errorf("BUILDKITE_AGENT_TAGS cannot override reserved tag %q", key)
			}
			continue
		}
		merged = append(merged, tag)
	}
	merged = append(merged, canonical...)
	return strings.Join(merged, ","), nil
}

func splitAgentTag(tag string) (string, string) {
	parts := strings.SplitN(tag, "=", 2)
	key := strings.TrimSpace(parts[0])
	value := ""
	if len(parts) == 2 {
		value = strings.TrimSpace(parts[1])
	}
	return key, value
}

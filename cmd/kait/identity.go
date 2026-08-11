package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const defaultIdentityPath = "/etc/kait/identity.json"

// identityPath is a variable so unit tests can exercise baked identity checks
// without writing to the host filesystem. Official images always use the
// fixed default path; it is not a runtime configuration knob.
var identityPath = defaultIdentityPath

type identity struct {
	Schema       int      `json:"schema"`
	Hardware     string   `json:"hardware"`
	Variant      string   `json:"variant"`
	Profile      string   `json:"profile"`
	Capabilities []string `json:"capabilities"`
	Requirements []string `json:"requirements,omitempty"`
}

func loadRuntimeIdentity() (identity, error) {
	data, err := os.ReadFile(identityPath)
	if errors.Is(err, os.ErrNotExist) {
		return identity{}, fmt.Errorf("Kait baked image identity is missing at %s", identityPath)
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
	id.Profile = strings.ToLower(strings.TrimSpace(id.Profile))
	if id.Profile == "" {
		id.Profile = profileForIdentity(id)
	}
	if err := validateIdentity(id); err != nil {
		return identity{}, err
	}

	if value := strings.TrimSpace(os.Getenv("KAIT_HARDWARE")); value != "" && strings.ToLower(value) != id.Hardware {
		return identity{}, fmt.Errorf("KAIT_HARDWARE=%q conflicts with baked image hardware %q", value, id.Hardware)
	}
	if value := strings.TrimSpace(os.Getenv("KAIT_VARIANT")); value != "" && strings.ToLower(value) != id.Variant {
		return identity{}, fmt.Errorf("KAIT_VARIANT=%q conflicts with baked image variant %q", value, id.Variant)
	}
	if value := strings.TrimSpace(os.Getenv("KAIT_PROFILE")); value != "" && strings.ToLower(value) != id.Profile {
		return identity{}, fmt.Errorf("KAIT_PROFILE=%q conflicts with baked image profile %q", value, id.Profile)
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

func validateHardware(hardware string) error {
	if _, ok := authoritativeCapabilities.Hardware[hardware]; !ok {
		return fmt.Errorf("KAIT_HARDWARE must be one of %s (got %q)", strings.Join(supportedHardwareNames(), ", "), hardware)
	}
	return nil
}

func validateVariant(variant string) error {
	if variant != "slim" && variant != "full" {
		return fmt.Errorf("KAIT_VARIANT must be one of slim, full (got %q)", variant)
	}
	return nil
}

func validateIdentity(id identity) error {
	if id.Schema != 1 && id.Schema != 2 {
		return fmt.Errorf("unsupported Kait image identity schema %d", id.Schema)
	}
	if err := validateHardware(id.Hardware); err != nil {
		return err
	}
	if err := validateVariant(id.Variant); err != nil {
		return err
	}
	profileDefinition, ok := authoritativeCapabilities.Profiles[id.Profile]
	if !ok {
		return fmt.Errorf("unsupported Kait image profile %q", id.Profile)
	}
	if profileDefinition.Variant != id.Variant {
		return fmt.Errorf("Kait image profile %q requires variant %q, got %q", id.Profile, profileDefinition.Variant, id.Variant)
	}
	if !sameCapabilities(profileDefinition.Capabilities, id.Capabilities) {
		return fmt.Errorf("Kait image identity capabilities %q do not match profile %q", strings.Join(id.Capabilities, ","), id.Profile)
	}
	if id.Schema == 2 && len(id.Requirements) == 0 {
		return fmt.Errorf("Kait image identity schema 2 must declare requirements")
	}
	if len(id.Requirements) > 0 {
		expected, err := orderedRequirements(id.Hardware, id.Profile)
		if err != nil {
			return err
		}
		if !sameStrings(expected, id.Requirements) {
			return fmt.Errorf("Kait image identity requirements do not match profile %q", id.Profile)
		}
	}
	return nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func canonicalAgentTags(id identity, o11y string) []string {
	tags := []string{
		"kait=true",
		"kait.hardware=" + id.Hardware,
		"kait.variant=" + id.Variant,
		"kait.profile=" + id.Profile,
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

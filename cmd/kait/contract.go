package main

import (
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
)

// The contract is embedded in the supervisor and is also used during image
// construction. This keeps the image identity, runtime checks, tags, and CI
// matrix on one versioned source of truth.
//
//go:embed capability-contract.json
var capabilityContractFS embed.FS

type hardwareDefinition struct {
	Runtime            string   `json:"runtime"`
	Platforms          []string `json:"platforms"`
	Python             string   `json:"python"`
	BaseImage          string   `json:"base_image"`
	Runner             string   `json:"runner"`
	Active             bool     `json:"active"`
	Accelerator        string   `json:"accelerator"`
	PythonRequirements string   `json:"python_requirements"`
	SupportedProfiles  []string `json:"supported_profiles"`
}

type capabilityDefinition struct {
	Requires             []string `json:"requires"`
	Requirements         []string `json:"requirements"`
	HardwareRequirements bool     `json:"hardware_requirements"`
	Summary              string   `json:"summary"`
	Smoke                string   `json:"smoke"`
}

type profileDefinition struct {
	Variant      string   `json:"variant"`
	Capabilities []string `json:"capabilities"`
}

type capabilityModel struct {
	Schema        int                             `json:"schema"`
	HardwareOrder []string                        `json:"hardware_order"`
	ProfileOrder  []string                        `json:"profile_order"`
	Hardware      map[string]hardwareDefinition   `json:"hardware"`
	Capabilities  map[string]capabilityDefinition `json:"capabilities"`
	Profiles      map[string]profileDefinition    `json:"profiles"`
}

var authoritativeCapabilities = mustLoadCapabilityModel()

func mustLoadCapabilityModel() capabilityModel {
	data, err := capabilityContractFS.ReadFile("capability-contract.json")
	if err != nil {
		panic(err)
	}
	var model capabilityModel
	if err := json.Unmarshal(data, &model); err != nil {
		panic(err)
	}
	if err := validateCapabilityModel(model); err != nil {
		panic(err)
	}
	return model
}

func validateCapabilityModel(model capabilityModel) error {
	if model.Schema != 1 {
		return fmt.Errorf("unsupported capability contract schema %d", model.Schema)
	}
	for _, hardware := range model.HardwareOrder {
		definition, ok := model.Hardware[hardware]
		if !ok {
			return fmt.Errorf("hardware order references undefined hardware %q", hardware)
		}
		if definition.Runtime == "" || len(definition.Platforms) == 0 || definition.Runner == "" || definition.Accelerator == "" {
			return fmt.Errorf("hardware %q must define runtime, platforms, runner, and accelerator", hardware)
		}
		switch definition.Runtime {
		case "container":
			if definition.BaseImage == "" {
				return fmt.Errorf("container hardware %q must define a base image", hardware)
			}
		case "native-macos":
			if definition.BaseImage != "" || len(definition.Platforms) != 1 || definition.Platforms[0] != "darwin/arm64" {
				return fmt.Errorf("native-macos hardware %q must target darwin/arm64 and have no base image", hardware)
			}
		default:
			return fmt.Errorf("hardware %q has unsupported runtime %q", hardware, definition.Runtime)
		}
		for _, profile := range definition.SupportedProfiles {
			if _, ok := model.Profiles[profile]; !ok {
				return fmt.Errorf("hardware %q references undefined supported profile %q", hardware, profile)
			}
		}
	}
	for _, profile := range model.ProfileOrder {
		definition, ok := model.Profiles[profile]
		if !ok {
			return fmt.Errorf("profile order references undefined profile %q", profile)
		}
		if _, ok := model.Profiles[profile]; !ok || definition.Variant == "" {
			return fmt.Errorf("profile %q has no compatibility variant", profile)
		}
		for _, capability := range definition.Capabilities {
			if _, ok := model.Capabilities[capability]; !ok {
				return fmt.Errorf("profile %q references undefined capability %q", profile, capability)
			}
		}
	}
	for capability, definition := range model.Capabilities {
		if strings.TrimSpace(definition.Summary) == "" || strings.TrimSpace(definition.Smoke) == "" {
			return fmt.Errorf("capability %q must define summary and smoke proof", capability)
		}
		for _, dependency := range definition.Requires {
			if _, ok := model.Capabilities[dependency]; !ok {
				return fmt.Errorf("capability %q requires undefined capability %q", capability, dependency)
			}
		}
	}
	return nil
}

func capabilityNames() []string {
	names := make([]string, 0, len(authoritativeCapabilities.Capabilities))
	for name := range authoritativeCapabilities.Capabilities {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func profileNames() []string {
	return append([]string(nil), authoritativeCapabilities.ProfileOrder...)
}

func supportedHardwareNames() []string {
	return append([]string(nil), authoritativeCapabilities.HardwareOrder...)
}

func profileForVariant(variant string) string {
	for _, profile := range authoritativeCapabilities.ProfileOrder {
		if authoritativeCapabilities.Profiles[profile].Variant == variant && (profile == "slim" || profile == "full") {
			return profile
		}
	}
	return ""
}

func profileForIdentity(id identity) string {
	if id.Profile != "" {
		return id.Profile
	}
	if profile := profileForVariant(id.Variant); profile != "" {
		for _, candidate := range authoritativeCapabilities.ProfileOrder {
			definition := authoritativeCapabilities.Profiles[candidate]
			if definition.Variant == id.Variant && sameCapabilities(definition.Capabilities, id.Capabilities) {
				return candidate
			}
		}
		return profile
	}
	return ""
}

func capabilitiesForProfile(profile string) []string {
	definition := authoritativeCapabilities.Profiles[profile]
	return append([]string(nil), definition.Capabilities...)
}

func capabilitiesForVariant(variant string) []string {
	if profile := profileForVariant(variant); profile != "" {
		return capabilitiesForProfile(profile)
	}
	return nil
}

func hardwareSupportsProfile(hardware, profile string) bool {
	definition, ok := authoritativeCapabilities.Hardware[hardware]
	if !ok {
		return false
	}
	for _, supported := range definition.SupportedProfiles {
		if supported == profile {
			return true
		}
	}
	return false
}

func orderedRequirements(hardware, profile string) ([]string, error) {
	hardwareDefinition, ok := authoritativeCapabilities.Hardware[hardware]
	if !ok {
		return nil, fmt.Errorf("unsupported Kait hardware %q", hardware)
	}
	profileDefinition, ok := authoritativeCapabilities.Profiles[profile]
	if !ok {
		return nil, fmt.Errorf("unsupported Kait profile %q", profile)
	}
	if !hardwareSupportsProfile(hardware, profile) {
		return nil, fmt.Errorf("Kait hardware %q does not support profile %q", hardware, profile)
	}
	requirements := make([]string, 0, 8)
	seenRequirements := make(map[string]bool)
	needsHardware := false
	seenCapabilities := make(map[string]bool)
	var visit func(string) error
	visit = func(capability string) error {
		if seenCapabilities[capability] {
			return nil
		}
		definition, ok := authoritativeCapabilities.Capabilities[capability]
		if !ok {
			return fmt.Errorf("unsupported Kait capability %q", capability)
		}
		for _, dependency := range definition.Requires {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		seenCapabilities[capability] = true
		if definition.HardwareRequirements {
			needsHardware = true
		}
		for _, requirement := range definition.Requirements {
			if !seenRequirements[requirement] {
				seenRequirements[requirement] = true
				requirements = append(requirements, requirement)
			}
		}
		return nil
	}
	for _, capability := range profileDefinition.Capabilities {
		if err := visit(capability); err != nil {
			return nil, err
		}
	}
	if needsHardware && hardwareDefinition.PythonRequirements != "" {
		requirements = append([]string{hardwareDefinition.PythonRequirements}, requirements...)
	}
	return requirements, nil
}

func resolveContract(hardware, profile, variant, declaredCapabilities string) (identity, error) {
	if hardware == "" {
		hardware = "cpu"
	}
	if profile == "" {
		if variant == "" {
			variant = "slim"
		}
		profile = profileForVariant(variant)
	}
	hardware = strings.ToLower(strings.TrimSpace(hardware))
	profile = strings.ToLower(strings.TrimSpace(profile))
	variant = strings.ToLower(strings.TrimSpace(variant))
	hardwareDefinition, ok := authoritativeCapabilities.Hardware[hardware]
	if !ok {
		return identity{}, fmt.Errorf("unsupported Kait hardware %q", hardware)
	}
	profileDefinition, ok := authoritativeCapabilities.Profiles[profile]
	if !ok {
		return identity{}, fmt.Errorf("unsupported Kait profile %q", profile)
	}
	if !hardwareSupportsProfile(hardware, profile) {
		return identity{}, fmt.Errorf("Kait hardware %q does not support profile %q", hardware, profile)
	}
	if variant == "" {
		variant = profileDefinition.Variant
	}
	if variant != profileDefinition.Variant {
		return identity{}, fmt.Errorf("Kait profile %q requires variant %q, got %q", profile, profileDefinition.Variant, variant)
	}
	capabilities := append([]string(nil), profileDefinition.Capabilities...)
	if declaredCapabilities != "" {
		declared, err := parseCapabilities(declaredCapabilities)
		if err != nil {
			return identity{}, err
		}
		if !sameCapabilities(declared, capabilities) {
			return identity{}, fmt.Errorf("declared capabilities %q do not match profile %q capabilities %q", declaredCapabilities, profile, strings.Join(capabilities, ","))
		}
	}
	requirements, err := orderedRequirements(hardware, profile)
	if err != nil {
		return identity{}, err
	}
	_ = hardwareDefinition
	return identity{
		Schema:       3,
		Hardware:     hardware,
		Runtime:      hardwareDefinition.Runtime,
		Accelerator:  hardwareDefinition.Accelerator,
		Variant:      variant,
		Profile:      profile,
		Capabilities: capabilities,
		Requirements: requirements,
	}, nil
}

func writeContract(w io.Writer, args []string) error {
	flags := flag.NewFlagSet("contract", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	hardware := flags.String("hardware", "cpu", "hardware class")
	profile := flags.String("profile", "", "workload profile")
	variant := flags.String("variant", "", "slim/full compatibility profile")
	capabilities := flags.String("capabilities", "", "legacy build-time capability assertion")
	if err := flags.Parse(args); err != nil {
		return err
	}
	contract, err := resolveContract(*hardware, *profile, *variant, *capabilities)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(contract)
}

type matrixEntry struct {
	Hardware     string   `json:"hardware"`
	Runtime      string   `json:"runtime"`
	Accelerator  string   `json:"accelerator"`
	Profile      string   `json:"profile"`
	Variant      string   `json:"variant"`
	Runner       string   `json:"runner"`
	Platform     string   `json:"platform"`
	Platforms    []string `json:"platforms"`
	BaseImage    string   `json:"base_image"`
	Python       string   `json:"python_bin"`
	Capabilities string   `json:"capabilities"`
	Target       string   `json:"target"`
	Active       bool     `json:"active"`
}

func writeMatrix(w io.Writer, args []string) error {
	flags := flag.NewFlagSet("matrix", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	activeOnly := flags.Bool("active-only", false, "include only active hardware")
	acceleratorsOnly := flags.Bool("accelerators-only", false, "include only inactive accelerator hardware")
	runtimeName := flags.String("runtime", "", "filter by execution runtime (container or native-macos)")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *activeOnly && *acceleratorsOnly {
		return errors.New("matrix cannot select both active-only and accelerators-only")
	}
	if *runtimeName != "" && *runtimeName != "container" && *runtimeName != "native-macos" {
		return fmt.Errorf("matrix runtime must be container or native-macos (got %q)", *runtimeName)
	}
	entries := make([]matrixEntry, 0)
	for _, hardware := range authoritativeCapabilities.HardwareOrder {
		definition := authoritativeCapabilities.Hardware[hardware]
		if *activeOnly && !definition.Active {
			continue
		}
		if *acceleratorsOnly && definition.Active {
			continue
		}
		if *runtimeName != "" && definition.Runtime != *runtimeName {
			continue
		}
		for _, profile := range authoritativeCapabilities.ProfileOrder {
			if !hardwareSupportsProfile(hardware, profile) {
				continue
			}
			profileDefinition := authoritativeCapabilities.Profiles[profile]
			entries = append(entries, matrixEntry{
				Hardware:     hardware,
				Runtime:      definition.Runtime,
				Accelerator:  definition.Accelerator,
				Profile:      profile,
				Variant:      profileDefinition.Variant,
				Runner:       definition.Runner,
				Platform:     definition.Platforms[0],
				Platforms:    definition.Platforms,
				BaseImage:    definition.BaseImage,
				Python:       definition.Python,
				Capabilities: strings.Join(profileDefinition.Capabilities, ","),
				Target:       hardware + "-" + profile,
				Active:       definition.Active,
			})
		}
	}
	return json.NewEncoder(w).Encode(map[string]any{"include": entries})
}

func parseCapabilities(value string) ([]string, error) {
	var capabilities []string
	seen := make(map[string]bool)
	for _, raw := range strings.Split(value, ",") {
		capability := strings.ToLower(strings.TrimSpace(raw))
		if capability == "" {
			return nil, errors.New("capability list contains an empty capability")
		}
		if _, ok := authoritativeCapabilities.Capabilities[capability]; !ok {
			return nil, fmt.Errorf("unsupported Kait capability %q", capability)
		}
		if seen[capability] {
			return nil, fmt.Errorf("duplicate Kait capability %q", capability)
		}
		seen[capability] = true
		capabilities = append(capabilities, capability)
	}
	if len(capabilities) == 0 {
		return nil, errors.New("capability list must not be empty")
	}
	return capabilities, nil
}

func smokeScripts(capabilities []string) (string, error) {
	var scripts []string
	seen := make(map[string]bool)
	for _, capability := range capabilities {
		if seen[capability] {
			continue
		}
		definition, ok := authoritativeCapabilities.Capabilities[capability]
		if !ok {
			return "", fmt.Errorf("unsupported Kait capability %q", capability)
		}
		seen[capability] = true
		scripts = append(scripts, definition.Smoke)
	}
	return strings.Join(scripts, "\n"), nil
}

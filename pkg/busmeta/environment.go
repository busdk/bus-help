// Package busmeta contains Bus namespaced metadata extensions embedded inside
// OpenCLI-compatible documents.
package busmeta

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/busdk/bus-help/pkg/opencli"
)

// Metadata key constants define the Bus OpenCLI profile namespace.
//
// Used by: bus-help, bus-configure, and module-owned metadata producers.
const (
	KeyProfile     = "io.busdk.profile"
	KeyEnvironment = "io.busdk.environment"
	KeyConfig      = "io.busdk.config"
	KeyEffects     = "io.busdk.effects"
	KeySecurity    = "io.busdk.security"
)

// ProfileMetadata identifies the Bus OpenCLI profile revision used by a document.
//
// Used by: module metadata constructors.
type ProfileMetadata struct {
	Version string `json:"version"`
	Module  string `json:"module,omitempty"`
}

// EnvironmentMetadata describes environment variables consumed by a Bus module.
//
// Used by: module metadata constructors and bus-configure.
type EnvironmentMetadata struct {
	Version        string                `json:"version"`
	Precedence     []string              `json:"precedence,omitempty"`
	Dotenv         []DotenvHint          `json:"dotenv,omitempty"`
	Variables      []EnvironmentVariable `json:"variables"`
	SourceModule   string                `json:"sourceModule,omitempty"`
	DiscoveryNotes []string              `json:"discoveryNotes,omitempty"`
}

// DotenvHint describes a conventional dotenv path for a module.
//
// Used by: EnvironmentMetadata.
type DotenvHint struct {
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`
}

// EnvironmentVariable describes one Bus environment variable contract.
//
// Used by: EnvironmentMetadata and bus-configure validation.
type EnvironmentVariable struct {
	Name         string         `json:"name"`
	Description  string         `json:"description,omitempty"`
	Required     bool           `json:"required,omitempty"`
	Secret       bool           `json:"secret,omitempty"`
	Generated    bool           `json:"generated,omitempty"`
	Default      string         `json:"default,omitempty"`
	Schema       Schema         `json:"schema,omitempty"`
	MapsTo       []Mapping      `json:"mapsTo,omitempty"`
	SafeHandling SafeHandling   `json:"safeHandling,omitempty"`
	Affects      []string       `json:"affects,omitempty"`
	Scope        string         `json:"scope,omitempty"`
	Examples     []ValueExample `json:"examples,omitempty"`
}

// Schema contains the small JSON Schema-style subset used for Bus env validation.
//
// Used by: EnvironmentVariable and ValidateValue.
type Schema struct {
	Type      string   `json:"type,omitempty"`
	Enum      []string `json:"enum,omitempty"`
	Format    string   `json:"format,omitempty"`
	MinLength int      `json:"minLength,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Default   string   `json:"default,omitempty"`
}

// Mapping connects an environment variable to another CLI or config surface.
//
// Used by: EnvironmentVariable.
type Mapping struct {
	Kind string `json:"kind,omitempty"`
	Name string `json:"name"`
}

// SafeHandling describes whether a value can be printed, persisted, logged, or generated.
//
// Used by: EnvironmentVariable.
type SafeHandling struct {
	Printable         bool `json:"printable"`
	StoreInDotenv     bool `json:"storeInDotenv"`
	RedactInLogs      bool `json:"redactInLogs"`
	GenerateSupported bool `json:"generateSupported,omitempty"`
}

// ValueExample gives a safe example value.
//
// Used by: EnvironmentVariable.
type ValueExample struct {
	Description string `json:"description,omitempty"`
	Value       string `json:"value,omitempty"`
}

// AttachEnvironment adds Bus profile and environment metadata to an OpenCLI document.
//
// Used by: module-owned metadata constructors.
func AttachEnvironment(doc *opencli.Document, module string, env EnvironmentMetadata) {
	if doc.Metadata == nil {
		doc.Metadata = map[string]any{}
	}
	if env.SourceModule == "" {
		env.SourceModule = module
	}
	doc.Metadata[KeyProfile] = ProfileMetadata{Version: "0.1", Module: module}
	doc.Metadata[KeyEnvironment] = env
}

// EnvironmentFromDocument extracts Bus environment metadata from an OpenCLI document.
//
// Used by: bus-help rendering and bus-configure discovery.
func EnvironmentFromDocument(doc opencli.Document) (EnvironmentMetadata, bool, error) {
	raw, ok := doc.Metadata[KeyEnvironment]
	if !ok {
		return EnvironmentMetadata{}, false, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return EnvironmentMetadata{}, true, fmt.Errorf("encode environment metadata: %w", err)
	}
	var env EnvironmentMetadata
	if err := json.Unmarshal(data, &env); err != nil {
		return EnvironmentMetadata{}, true, fmt.Errorf("decode environment metadata: %w", err)
	}
	return env, true, ValidateEnvironment(env)
}

// ValidateEnvironment validates the Bus environment metadata shape.
//
// Used by: unit tests, bus-help rendering, and bus-configure discovery.
func ValidateEnvironment(env EnvironmentMetadata) error {
	if env.Version == "" {
		return fmt.Errorf("environment metadata version is required")
	}
	seen := map[string]bool{}
	for _, variable := range env.Variables {
		if variable.Name == "" {
			return fmt.Errorf("environment variable name is required")
		}
		if seen[variable.Name] {
			return fmt.Errorf("duplicate environment variable: %s", variable.Name)
		}
		seen[variable.Name] = true
		if err := validateVariableName(variable.Name); err != nil {
			return err
		}
		if variable.Schema.Pattern != "" {
			if _, err := regexp.Compile(variable.Schema.Pattern); err != nil {
				return fmt.Errorf("%s has invalid pattern: %w", variable.Name, err)
			}
		}
	}
	return nil
}

// ValidateValue validates one environment value against the supported schema subset.
//
// Used by: bus-configure validate and doctor commands.
func ValidateValue(variable EnvironmentVariable, value string) error {
	if variable.Required && value == "" {
		return fmt.Errorf("%s is required", variable.Name)
	}
	if value == "" {
		return nil
	}
	schema := variable.Schema
	if schema.Type != "" && schema.Type != "string" {
		return fmt.Errorf("%s has unsupported schema type %q", variable.Name, schema.Type)
	}
	if schema.MinLength > 0 && len(value) < schema.MinLength {
		return fmt.Errorf("%s must be at least %d characters", variable.Name, schema.MinLength)
	}
	if len(schema.Enum) > 0 {
		for _, allowed := range schema.Enum {
			if value == allowed {
				return nil
			}
		}
		return fmt.Errorf("%s must be one of %s", variable.Name, strings.Join(schema.Enum, ","))
	}
	if schema.Pattern != "" {
		matched, err := regexp.MatchString(schema.Pattern, value)
		if err != nil {
			return fmt.Errorf("%s has invalid pattern: %w", variable.Name, err)
		}
		if !matched {
			return fmt.Errorf("%s does not match required pattern", variable.Name)
		}
	}
	return nil
}

// RedactedValue returns a display-safe value for an environment variable.
//
// Used by: bus-configure list and doctor output.
func RedactedValue(variable EnvironmentVariable, value string) string {
	if value == "" {
		return ""
	}
	if variable.Secret || variable.SafeHandling.RedactInLogs || !variable.SafeHandling.Printable {
		return "********"
	}
	return value
}

func validateVariableName(name string) error {
	for i, r := range name {
		valid := r == '_' || ('A' <= r && r <= 'Z') || (i > 0 && '0' <= r && r <= '9')
		if !valid {
			return fmt.Errorf("invalid environment variable name: %s", name)
		}
	}
	return nil
}

package modulehelp

import (
	"strings"

	"github.com/busdk/bus-help/pkg/busmeta"
	"github.com/busdk/bus-help/pkg/opencli"
)

// EnvVar describes a module-owned environment variable for simple OpenCLI metadata.
//
// Used by: Bus module binaries that do not need custom OpenCLI construction.
type EnvVar struct {
	Name          string
	Description   string
	Required      bool
	Secret        bool
	Default       string
	Enum          []string
	Pattern       string
	Format        string
	Affects       []string
	Scope         string
	StoreInDotenv bool
}

// SimpleDocument returns a deterministic OpenCLI document for a Bus module.
//
// Used by: module-owned metadata adapters in Bus command binaries.
func SimpleDocument(module string, binary string, summary string, variables []EnvVar) opencli.Document {
	if binary == "" {
		binary = "bus-" + module
	}
	doc := opencli.Document{
		OpenCLI: "0.1.0",
		Info: opencli.Info{
			Title:       binary,
			Version:     "dev",
			Summary:     summary,
			Description: summary,
		},
		Commands: []opencli.Command{{
			Name:    "command",
			Summary: summary,
			Usage:   binary + " [options] [command...]",
			Options: []opencli.Option{
				{Name: "--help", Aliases: []string{"-h"}, Description: "Show human-readable help and exit."},
				{Name: "--version", Aliases: []string{"-V"}, Description: "Print version information and exit."},
				{Name: "--format", Aliases: []string{"-f"}, ValueName: "format", Description: "Select output format where supported."},
			},
			Examples: []opencli.Example{
				{Summary: "Show human help.", Command: binary + " --help"},
				{Summary: "Show machine-readable metadata.", Command: binary + " help --format opencli"},
			},
			ExitCodes: []opencli.ExitCode{
				{Code: 0, Description: "Success."},
				{Code: 1, Description: "Runtime error."},
				{Code: 2, Description: "Usage error."},
			},
		}},
	}
	env := busmeta.EnvironmentMetadata{
		Version:    "0.1",
		Precedence: []string{"CLI flags", ".env", "process environment", "module defaults"},
		Dotenv:     []busmeta.DotenvHint{{Path: ".env", Description: "Workspace or deployment-local Bus environment."}},
		Variables:  make([]busmeta.EnvironmentVariable, 0, len(variables)),
	}
	for _, variable := range variables {
		env.Variables = append(env.Variables, variable.toBusMeta())
	}
	busmeta.AttachEnvironment(&doc, module, env)
	return doc
}

// SimpleTextHelp returns concise text for the module-local help subcommand.
//
// Used by: module-owned metadata adapters in Bus command binaries.
func SimpleTextHelp(binary string, summary string) string {
	return binary + " exposes live Bus help metadata.\n\nUsage:\n  " + binary + " help [--format text|opencli|json]\n\n" + summary + "\n"
}

func (v EnvVar) toBusMeta() busmeta.EnvironmentVariable {
	safe := busmeta.SafeHandling{
		Printable:     !v.Secret,
		StoreInDotenv: true,
		RedactInLogs:  v.Secret,
	}
	if !v.StoreInDotenv && v.StoreInDotenvSet() {
		safe.StoreInDotenv = false
	}
	schema := busmeta.Schema{Type: "string", Default: v.Default, Format: v.Format, Enum: v.Enum, Pattern: v.Pattern}
	scope := v.Scope
	if scope == "" {
		scope = "deployment"
	}
	affects := v.Affects
	if len(affects) == 0 {
		affects = []string{inferAffects(v.Name)}
	}
	return busmeta.EnvironmentVariable{
		Name:         v.Name,
		Description:  v.description(),
		Required:     v.Required,
		Secret:       v.Secret,
		Default:      v.Default,
		Schema:       schema,
		SafeHandling: safe,
		Affects:      affects,
		Scope:        scope,
	}
}

// description returns module-provided or standard purpose text for an env var.
//
// Used by: EnvVar OpenCLI conversion.
func (v EnvVar) description() string {
	description := strings.TrimSpace(v.Description)
	if description == "" || description == v.Name+" setting used by this Bus module." {
		if standard := standardDescription(v.Name); standard != "" {
			return standard
		}
		return ""
	}
	return description
}

// StoreInDotenvSet reports whether StoreInDotenv was intentionally set false.
//
// Used by: EnvVar conversion.
func (v EnvVar) StoreInDotenvSet() bool {
	return v.Secret && strings.Contains(v.Description, "must not be stored")
}

func inferAffects(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "key") || strings.Contains(lower, "password"):
		return "security"
	case strings.Contains(lower, "database") || strings.Contains(lower, "postgres") || strings.Contains(lower, "dsn"):
		return "storage"
	case strings.Contains(lower, "api") || strings.Contains(lower, "url"):
		return "api"
	default:
		return "configuration"
	}
}

// standardDescription returns shared purpose text for common Bus environment variables.
//
// Used by: EnvVar.description.
func standardDescription(name string) string {
	switch {
	case name == "BUS_DEV":
		return "Path or command used to invoke the local bus-dev helper during module development checks."
	case name == "BUS_E2E_KEEP":
		return "Keep temporary e2e workspaces after a test run for debugging."
	case name == "BUS_E2E_VERBOSE":
		return "Enable verbose e2e script logging for troubleshooting."
	case name == "BUS_GO_QUALITY_PROFILE":
		return "bus-dev quality lint profile used by this module's Makefile lint target."
	case name == "BUS_API_TOKEN":
		return "Bus API bearer token used for authenticated API or worker requests."
	case name == "BUS_EVENTS_API_URL":
		return "Base URL for the Bus Events API used by event-backed integrations and providers."
	case name == "BUS_AUTH_API_URL":
		return "Base URL for the Bus Auth API used to issue or validate Bus credentials."
	case name == "BUS_AI_API_URL":
		return "Base URL for the Bus AI API used by AI-facing clients and providers."
	case name == "BUS_API_URL":
		return "Base URL for the Bus API used by this module."
	case strings.HasSuffix(name, "_JWT_SECRET"):
		return "HS256 JWT signing or verification secret for this module's protected API surface."
	case strings.HasSuffix(name, "_HS256_SECRET"):
		return "HS256 JWT signing or verification secret for local Bus authentication."
	case strings.Contains(name, "POSTGRES_DSN") || strings.Contains(name, "DATABASE_URL"):
		return "Database connection string used by this module's persistent backend."
	case strings.Contains(name, "TOKEN"):
		return "Authentication token used by this module."
	case strings.Contains(name, "SECRET"):
		return "Secret value used by this module."
	case strings.Contains(name, "API_URL") || strings.HasSuffix(name, "_URL"):
		return "Service URL used by this module."
	default:
		return ""
	}
}

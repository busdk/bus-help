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
		Description:  v.Description,
		Required:     v.Required,
		Secret:       v.Secret,
		Default:      v.Default,
		Schema:       schema,
		SafeHandling: safe,
		Affects:      affects,
		Scope:        scope,
	}
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

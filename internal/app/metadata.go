package app

import (
	"github.com/busdk/bus-help/pkg/busmeta"
	"github.com/busdk/bus-help/pkg/opencli"
)

// openCLIDocument returns bus-help's module-owned OpenCLI metadata document.
//
// Used by: Run when handling `help --format opencli`.
func openCLIDocument() opencli.Document {
	doc := opencli.Document{
		OpenCLI: "0.1.0",
		Info: opencli.Info{
			Title:       "bus-help",
			Version:     "dev",
			Summary:     "Render Bus help and live machine-readable metadata.",
			Description: "Discovers module-owned OpenCLI metadata from live Bus module help commands and renders human or structured help.",
		},
		Commands: []opencli.Command{
			{
				Name:    "module",
				Summary: "Render help metadata for one module.",
				Usage:   "bus-help [--format text|opencli|json] MODULE [COMMAND...]",
				Arguments: []opencli.Argument{
					{Name: "MODULE", Required: true, Description: "Bus module name without the bus- prefix."},
					{Name: "COMMAND", Repeatable: true, Description: "Optional command path within the module."},
				},
				Options: commonOptions(),
				Examples: []opencli.Example{
					{Summary: "Show human help for a module.", Command: "bus-help journal"},
					{Summary: "Show OpenCLI metadata for a module.", Command: "bus-help --format opencli journal"},
				},
				ExitCodes: exitCodes(),
			},
			{
				Name:    "env",
				Summary: "Render Bus environment metadata for one module.",
				Usage:   "bus-help env MODULE",
				Arguments: []opencli.Argument{
					{Name: "MODULE", Required: true, Description: "Bus module name without the bus- prefix."},
				},
				Examples:  []opencli.Example{{Summary: "Show environment variables for journal.", Command: "bus-help env journal"}},
				ExitCodes: exitCodes(),
			},
			{
				Name:    "config",
				Summary: "Render Bus configuration metadata for one module.",
				Usage:   "bus-help config MODULE",
				Arguments: []opencli.Argument{
					{Name: "MODULE", Required: true, Description: "Bus module name without the bus- prefix."},
				},
				Examples:  []opencli.Example{{Summary: "Show configuration metadata for journal.", Command: "bus-help config journal"}},
				ExitCodes: exitCodes(),
			},
		},
	}
	busmeta.AttachEnvironment(&doc, "help", busmeta.EnvironmentMetadata{
		Version:    "0.1",
		Precedence: []string{".env", "process environment", "module defaults"},
		Dotenv:     []busmeta.DotenvHint{{Path: ".env", Description: "Workspace environment consumed by discovered modules."}},
		Variables:  []busmeta.EnvironmentVariable{},
	})
	return doc
}

func commonOptions() []opencli.Option {
	return []opencli.Option{
		{Name: "--format", Aliases: []string{"-f"}, ValueName: "text|opencli|json", Description: "Select output format.", Default: "text"},
		{Name: "--help", Aliases: []string{"-h"}, Description: "Show human-readable help and exit."},
	}
}

func exitCodes() []opencli.ExitCode {
	return []opencli.ExitCode{
		{Code: 0, Description: "Success."},
		{Code: 1, Description: "Metadata discovery or rendering failed."},
		{Code: 2, Description: "Usage error."},
	}
}

// Package app implements the bus-help CLI as a thin wrapper over metadata discovery.
package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/busdk/bus-help/pkg/busmeta"
	"github.com/busdk/bus-help/pkg/discovery"
	"github.com/busdk/bus-help/pkg/modulehelp"
)

// Run executes bus-help and returns the intended process exit code.
//
// Used by: cmd/bus-help/main.go and CLI tests.
func Run(args []string, workdir string, stdout io.Writer, stderr io.Writer) int {
	if handled, code := modulehelp.Handle(args[1:], stdout, stderr, helpText(), openCLIDocument()); handled {
		return code
	}
	flags, rest, err := parse(args[1:])
	if err != nil {
		writeUsageError(stderr, err.Error())
		return 2
	}
	if flags.help {
		writeHelp(stdout)
		return 0
	}
	if flags.format == "" {
		flags.format = "text"
	}
	if flags.format != "text" && flags.format != "opencli" && flags.format != "json" {
		writeUsageError(stderr, "unsupported format: "+flags.format)
		return 2
	}
	if len(rest) == 0 {
		if flags.format != "text" {
			writeUsageError(stderr, "module is required for structured output")
			return 2
		}
		writeHelp(stdout)
		return 0
	}

	switch rest[0] {
	case "env":
		if len(rest) != 2 {
			writeUsageError(stderr, "usage: bus-help env MODULE")
			return 2
		}
		return renderEnvironment(workdir, rest[1], flags.format, stdout, stderr)
	case "config":
		if len(rest) != 2 {
			writeUsageError(stderr, "usage: bus-help config MODULE")
			return 2
		}
		return renderEnvironment(workdir, rest[1], flags.format, stdout, stderr)
	default:
		module := rest[0]
		result := discovery.New(workdir).DiscoverModule(context.Background(), module)
		writeWarnings(stderr, result.Warnings)
		if !result.Found {
			if flags.format == "text" {
				fmt.Fprintf(stdout, "No structured help metadata available for %s.\n", module)
				return 0
			}
			return 1
		}
		if flags.format == "text" {
			fmt.Fprintf(stdout, "%s\n\n%s\n", result.Document.Info.Title, result.Document.Info.Summary)
			if env, ok, _ := busmeta.EnvironmentFromDocument(result.Document); ok {
				fmt.Fprintf(stdout, "\nEnvironment variables: %d\n", len(env.Variables))
			}
			return 0
		}
		return writeJSON(stdout, result.Document)
	}
}

type flags struct {
	format string
	help   bool
}

func parse(args []string) (flags, []string, error) {
	var out flags
	rest := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			out.help = true
		case arg == "-f" || arg == "--format":
			if i+1 >= len(args) {
				return flags{}, nil, fmt.Errorf("missing value for %s", arg)
			}
			i++
			out.format = args[i]
		case strings.HasPrefix(arg, "--format="):
			out.format = strings.TrimPrefix(arg, "--format=")
		case strings.HasPrefix(arg, "-"):
			return flags{}, nil, fmt.Errorf("unknown flag: %s", arg)
		default:
			rest = append(rest, arg)
		}
	}
	return out, rest, nil
}

func renderEnvironment(workdir string, module string, format string, stdout io.Writer, stderr io.Writer) int {
	result := discovery.New(workdir).DiscoverModule(context.Background(), module)
	writeWarnings(stderr, result.Warnings)
	if !result.Found {
		return 1
	}
	env, ok, err := busmeta.EnvironmentFromDocument(result.Document)
	if err != nil {
		writeUsageError(stderr, err.Error())
		return 1
	}
	if !ok {
		fmt.Fprintf(stdout, "No Bus environment metadata available for %s.\n", module)
		return 0
	}
	sort.SliceStable(env.Variables, func(i, j int) bool {
		return env.Variables[i].Name < env.Variables[j].Name
	})
	if format == "opencli" || format == "json" {
		return writeJSON(stdout, env)
	}
	fmt.Fprintf(stdout, "Environment metadata for %s\n", module)
	for _, variable := range env.Variables {
		required := "optional"
		if variable.Required {
			required = "required"
		}
		secret := "plain"
		if variable.Secret {
			secret = "secret"
		}
		fmt.Fprintf(stdout, "%s\t%s\t%s\t%s\n", variable.Name, required, secret, variable.Description)
	}
	return 0
}

func writeJSON(w io.Writer, value any) int {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return 1
	}
	return 0
}

func writeWarnings(w io.Writer, warnings []discovery.Warning) {
	for _, warning := range warnings {
		if warning.Message == "" {
			continue
		}
		if warning.Command == "" {
			fmt.Fprintf(w, "bus-help: warning: %s: %s\n", warning.Module, warning.Message)
			continue
		}
		fmt.Fprintf(w, "bus-help: warning: %s: %s\n", warning.Command, warning.Message)
	}
}

func writeHelp(w io.Writer) {
	io.WriteString(w, helpText())
}

func helpText() string {
	return `bus-help renders Bus command help and live machine-readable metadata.

Usage:
  bus-help [--format text|opencli|json] MODULE [COMMAND...]
  bus-help env MODULE
  bus-help config MODULE

Examples:
  bus-help journal
  bus-help --format opencli journal
  bus-help env journal
`
}

func writeUsageError(w io.Writer, msg string) {
	fmt.Fprintf(w, "bus-help: %s\n", msg)
}

// Main exits after running the CLI.
//
// Used by: cmd/bus-help/main.go.
func Main(args []string) {
	workdir, err := os.Getwd()
	if err != nil {
		os.Stderr.WriteString("bus-help: failed to determine working directory\n")
		os.Exit(1)
	}
	os.Exit(Run(args, workdir, os.Stdout, os.Stderr))
}

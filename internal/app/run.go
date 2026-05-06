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
	"github.com/busdk/bus-help/pkg/diagnostics"
	"github.com/busdk/bus-help/pkg/discovery"
	"github.com/busdk/bus-help/pkg/modulehelp"
)

// newDiscoverer constructs live metadata discovery for the CLI.
//
// Used by: Run and tests that inject deterministic discovery runners.
var newDiscoverer = discovery.New

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
	log := logger{level: flags.diagnosticLevel, writer: stderr}
	if len(rest) > 0 && rest[0] == "help" {
		if len(rest) != 1 {
			writeUsageError(stderr, "usage: bus-help help")
			return 2
		}
		if flags.format == "text" {
			writeHelp(stdout)
			return 0
		}
		return writeJSON(stdout, openCLIDocument())
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
		return renderEnvironment(workdir, rest[1], flags.format, stdout, log)
	case "config":
		if len(rest) != 2 {
			writeUsageError(stderr, "usage: bus-help config MODULE")
			return 2
		}
		return renderEnvironment(workdir, rest[1], flags.format, stdout, log)
	default:
		module := rest[0]
		log.Debug("discovering module metadata: module=%s", module)
		log.Trace("discovery context: workdir=%s format=%s", workdir, flags.format)
		result := newDiscoverer(workdir).DiscoverModule(context.Background(), module)
		writeWarnings(log, result.Warnings)
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

// flags stores parsed bus-help global flags.
//
// Used by: parse and Run.
type flags struct {
	format          string
	help            bool
	diagnosticFlags diagnostics.Flags
	diagnosticLevel diagnostics.Level
}

// parse converts CLI arguments into global flags and positional command args.
//
// Used by: Run.
func parse(args []string) (flags, []string, error) {
	var out flags
	rest := make([]string, 0, len(args))
	args = diagnostics.ExpandVerbosityArgs(args)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--":
			rest = append(rest, args[i+1:]...)
			i = len(args)
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
		case arg == "-v" || arg == "--verbose":
			if err := out.diagnosticFlags.Verbose.Set("true"); err != nil {
				return flags{}, nil, err
			}
		case strings.HasPrefix(arg, "--verbose="):
			if err := out.diagnosticFlags.Verbose.Set(strings.TrimPrefix(arg, "--verbose=")); err != nil {
				return flags{}, nil, err
			}
		case arg == "-q" || arg == "--quiet":
			out.diagnosticFlags.Quiet = true
		case strings.HasPrefix(arg, "--quiet="):
			value := strings.TrimPrefix(arg, "--quiet=")
			switch value {
			case "true":
				out.diagnosticFlags.Quiet = true
			case "false":
				out.diagnosticFlags.Quiet = false
			default:
				return flags{}, nil, fmt.Errorf("invalid quiet value: %s", value)
			}
		case arg == "--trace":
			out.diagnosticFlags.Trace = true
		case strings.HasPrefix(arg, "--trace="):
			value := strings.TrimPrefix(arg, "--trace=")
			switch value {
			case "true":
				out.diagnosticFlags.Trace = true
			case "false":
				out.diagnosticFlags.Trace = false
			default:
				return flags{}, nil, fmt.Errorf("invalid trace value: %s", value)
			}
		case strings.HasPrefix(arg, "-"):
			return flags{}, nil, fmt.Errorf("unknown flag: %s", arg)
		default:
			rest = append(rest, arg)
		}
	}
	level, err := out.diagnosticFlags.Level()
	if err != nil {
		return flags{}, nil, err
	}
	out.diagnosticLevel = level
	return out, rest, nil
}

// renderEnvironment writes environment/config metadata for one discovered module.
//
// Used by: Run env and config command paths.
func renderEnvironment(workdir string, module string, format string, stdout io.Writer, log logger) int {
	log.Debug("discovering environment metadata: module=%s", module)
	log.Trace("environment discovery context: workdir=%s format=%s", workdir, format)
	result := newDiscoverer(workdir).DiscoverModule(context.Background(), module)
	writeWarnings(log, result.Warnings)
	if !result.Found {
		return 1
	}
	env, ok, err := busmeta.EnvironmentFromDocument(result.Document)
	if err != nil {
		log.Error(err.Error())
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

// writeJSON writes indented JSON and returns a process-style status.
//
// Used by: Run and renderEnvironment.
func writeJSON(w io.Writer, value any) int {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		return 1
	}
	return 0
}

// writeWarnings emits discovery warnings through the diagnostic logger.
//
// Used by: Run and renderEnvironment.
func writeWarnings(log logger, warnings []discovery.Warning) {
	for _, warning := range warnings {
		if warning.Message == "" {
			continue
		}
		if warning.Command == "" {
			log.Warn("%s: %s", warning.Module, warning.Message)
			continue
		}
		log.Warn("%s: %s", warning.Command, warning.Message)
	}
}

// writeHelp writes human-readable bus-help usage.
//
// Used by: Run.
func writeHelp(w io.Writer) {
	io.WriteString(w, helpText())
}

// helpText returns the human-readable bus-help usage text.
//
// Used by: Run, modulehelp.Handle, and OpenCLI examples.
func helpText() string {
	return `bus-help renders Bus command help and live machine-readable metadata.

Usage:
  bus-help [--format text|opencli|json] [-v|-vv|--trace|--quiet] MODULE [COMMAND...]
  bus-help [--format text|opencli|json] [-v|-vv|--trace|--quiet] env MODULE
  bus-help [--format text|opencli|json] [-v|-vv|--trace|--quiet] config MODULE

Examples:
  bus-help journal
  bus-help --format opencli journal
  bus-help env journal

Diagnostics:
  default INFO, -v/--verbose DEBUG, -vv/repeated verbose/--trace TRACE, --quiet ERROR-only
`
}

// writeUsageError writes a concise usage diagnostic.
//
// Used by: Run and parse callers.
func writeUsageError(w io.Writer, msg string) {
	fmt.Fprintf(w, "bus-help: %s\n", msg)
}

// logger writes standard Bus diagnostic messages to stderr-like output.
//
// Used by: Run and renderEnvironment.
type logger struct {
	level  diagnostics.Level
	writer io.Writer
}

// Error writes an ERROR diagnostic.
//
// Used by: renderEnvironment.
func (l logger) Error(format string, args ...any) {
	l.log(diagnostics.LevelError, format, args...)
}

// Warn writes a WARN diagnostic.
//
// Used by: writeWarnings.
func (l logger) Warn(format string, args ...any) {
	l.log(diagnostics.LevelWarn, format, args...)
}

// Debug writes a DEBUG diagnostic.
//
// Used by: Run and renderEnvironment.
func (l logger) Debug(format string, args ...any) {
	l.log(diagnostics.LevelDebug, format, args...)
}

// Trace writes a TRACE diagnostic.
//
// Used by: Run and renderEnvironment.
func (l logger) Trace(format string, args ...any) {
	l.log(diagnostics.LevelTrace, format, args...)
}

// log writes one diagnostic when enabled by the parsed threshold.
//
// Used by: logger severity helpers.
func (l logger) log(level diagnostics.Level, format string, args ...any) {
	if l.writer == nil || !l.level.Enabled(level) {
		return
	}
	fmt.Fprintf(l.writer, "bus-help: %s: %s\n", level, fmt.Sprintf(format, args...))
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

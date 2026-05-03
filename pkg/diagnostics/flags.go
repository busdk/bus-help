// Package diagnostics provides shared Bus CLI diagnostic flag helpers.
package diagnostics

import (
	"flag"
	"fmt"
	"strconv"
)

// Level is the standard Bus diagnostic severity threshold.
//
// INFO should describe meaningful actions and affected entities without
// sensitive values. WARN describes abnormal but possibly recoverable behavior.
// ERROR describes clear failures. DEBUG adds verbose bug-finding detail. TRACE
// is exhaustive and may include sensitive values, so it must not be enabled in
// production or live environments.
//
// Used by: Bus CLI modules that implement TRACE, DEBUG, INFO, WARN, and ERROR
// diagnostics consistently.
type Level int

const (
	// LevelError prints only clear failures.
	//
	// Used by: --quiet behavior in Bus CLI modules.
	LevelError Level = iota
	// LevelWarn prints abnormal-but-possibly-recoverable warnings and errors.
	//
	// Used by: modules that expose warning-only diagnostics.
	LevelWarn
	// LevelInfo prints concise action/entity diagnostics for normal operation.
	//
	// Used by: default Bus CLI diagnostic behavior.
	LevelInfo
	// LevelDebug prints verbose testing and bug-finding diagnostics.
	//
	// Used by: one -v/--verbose flag.
	LevelDebug
	// LevelTrace prints exhaustive diagnostics and may expose sensitive values.
	//
	// Used by: -vv, repeated --verbose, or --trace.
	LevelTrace
)

// String returns the stable uppercase level name.
//
// Used by: diagnostics and tests.
func (l Level) String() string {
	switch l {
	case LevelError:
		return "ERROR"
	case LevelWarn:
		return "WARN"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	case LevelTrace:
		return "TRACE"
	default:
		return fmt.Sprintf("LEVEL(%d)", int(l))
	}
}

// Enabled reports whether a diagnostic at want should be printed.
//
// Used by: module logging adapters.
func (l Level) Enabled(want Level) bool {
	return l >= want
}

// Flags stores standard Bus diagnostic flags.
//
// Used by: Bus CLI modules while parsing global or command-local flags.
type Flags struct {
	Verbose Verbosity
	Quiet   bool
	Trace   bool
}

// AddFlags registers standard diagnostic flags on fs.
//
// Used by: Bus CLI modules that use the standard library flag package.
func AddFlags(fs *flag.FlagSet, flags *Flags) {
	fs.Var(&flags.Verbose, "v", "increase diagnostic verbosity; repeat for TRACE")
	fs.Var(&flags.Verbose, "verbose", "increase diagnostic verbosity; repeat for TRACE")
	fs.BoolVar(&flags.Quiet, "q", false, "suppress non-error diagnostics")
	fs.BoolVar(&flags.Quiet, "quiet", false, "suppress non-error diagnostics")
	fs.BoolVar(&flags.Trace, "trace", false, "enable TRACE diagnostics; equivalent to -vv")
}

// Level returns the standard diagnostic level for the parsed flags.
//
// Used by: command setup after flag parsing.
func (f Flags) Level() (Level, error) {
	return LevelFor(f.Quiet, f.Trace, f.Verbose.Count)
}

// LevelFor maps parsed verbosity controls to a standard diagnostic level.
//
// Used by: Flags.Level and tests.
func LevelFor(quiet bool, trace bool, verboseCount int) (Level, error) {
	if quiet && verboseCount > 0 {
		return LevelError, fmt.Errorf("--quiet and --verbose are mutually exclusive")
	}
	if quiet {
		return LevelError, nil
	}
	if trace || verboseCount > 1 {
		return LevelTrace, nil
	}
	if verboseCount == 1 {
		return LevelDebug, nil
	}
	return LevelInfo, nil
}

// Verbosity counts repeated -v/--verbose flags.
//
// Used by: Flags and module flag parsing.
type Verbosity struct {
	Count int
}

// String returns the parsed verbosity count.
//
// Used by: flag.Value.
func (v *Verbosity) String() string {
	return strconv.Itoa(v.Count)
}

// Set increments verbosity for boolean-style flags.
//
// Used by: flag.Value.
func (v *Verbosity) Set(value string) error {
	if value == "" || value == "true" {
		v.Count++
		return nil
	}
	if value == "false" {
		return nil
	}
	for _, r := range value {
		if r != 'v' {
			return fmt.Errorf("invalid verbosity %q", value)
		}
		v.Count++
	}
	return nil
}

// IsBoolFlag lets `flag` parse -v and --verbose without an explicit value.
//
// Used by: flag package.
func (v *Verbosity) IsBoolFlag() bool {
	return true
}

// ExpandVerbosityArgs lets the standard flag package accept compact -vv.
//
// Used by: modules before flag parsing.
func ExpandVerbosityArgs(args []string) []string {
	out := make([]string, 0, len(args))
	passthrough := false
	for _, arg := range args {
		if passthrough {
			out = append(out, arg)
			continue
		}
		if arg == "--" {
			passthrough = true
			out = append(out, arg)
			continue
		}
		if len(arg) > 2 && arg[0] == '-' && arg[1] != '-' {
			allVerbose := true
			for _, r := range arg[1:] {
				if r != 'v' {
					allVerbose = false
					break
				}
			}
			if allVerbose {
				for range arg[1:] {
					out = append(out, "-v")
				}
				continue
			}
		}
		out = append(out, arg)
	}
	return out
}

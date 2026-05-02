// Package discovery finds module-owned live metadata by executing Bus module help
// commands with explicit argv arrays.
package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/busdk/bus-help/pkg/opencli"
)

// Warning describes a non-fatal metadata discovery problem.
//
// Used by: Result and callers that continue after one module fails.
type Warning struct {
	Module  string `json:"module"`
	Command string `json:"command,omitempty"`
	Message string `json:"message"`
}

// Result contains one module discovery attempt.
//
// Used by: bus-help and bus-configure.
type Result struct {
	Module   string
	Document opencli.Document
	Found    bool
	Warnings []Warning
}

// Runner executes a command and returns captured stdout and stderr separately.
//
// Used by: Discoverer; tests provide fakes and production uses ExecRunner.
type Runner interface {
	Run(ctx context.Context, name string, args []string, dir string) ([]byte, []byte, int, error)
}

// ExecRunner is the production Runner backed by os/exec.
//
// Used by: New.
type ExecRunner struct{}

// Run executes one command without a shell and captures stdout/stderr.
//
// Used by: Discoverer.
func (ExecRunner) Run(ctx context.Context, name string, args []string, dir string) ([]byte, []byte, int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		exitCode = 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	return stdout.Bytes(), stderr.Bytes(), exitCode, err
}

// Discoverer configures live metadata discovery.
//
// Used by: bus-help and bus-configure app packages.
type Discoverer struct {
	Runner     Runner
	BusCommand string
	Timeout    time.Duration
	Workdir    string
}

// New creates a production Discoverer.
//
// Used by: CLI app constructors.
func New(workdir string) Discoverer {
	return Discoverer{
		Runner:     ExecRunner{},
		BusCommand: "bus",
		Timeout:    2 * time.Second,
		Workdir:    workdir,
	}
}

// DiscoverModule fetches one module's OpenCLI metadata through live command output.
//
// Used by: bus-help and bus-configure command handlers.
func (d Discoverer) DiscoverModule(ctx context.Context, module string) Result {
	result := Result{Module: module}
	if err := validateModuleName(module); err != nil {
		result.Warnings = append(result.Warnings, Warning{Module: module, Message: err.Error()})
		return result
	}
	runner := d.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	timeout := d.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	attempts := d.attempts(module)
	for _, attempt := range attempts {
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		stdout, stderr, exitCode, err := runner.Run(attemptCtx, attempt.name, attempt.args, d.Workdir)
		cancel()
		commandText := strings.TrimSpace(attempt.name + " " + strings.Join(attempt.args, " "))
		if attemptCtx.Err() == context.DeadlineExceeded {
			result.Warnings = append(result.Warnings, Warning{Module: module, Command: commandText, Message: "metadata command timed out"})
			continue
		}
		if err != nil || exitCode != 0 {
			msg := strings.TrimSpace(string(stderr))
			if msg == "" && err != nil {
				msg = err.Error()
			}
			result.Warnings = append(result.Warnings, Warning{Module: module, Command: commandText, Message: msg})
			continue
		}
		var doc opencli.Document
		if err := json.Unmarshal(stdout, &doc); err != nil {
			result.Warnings = append(result.Warnings, Warning{Module: module, Command: commandText, Message: fmt.Sprintf("invalid OpenCLI JSON: %s", err)})
			continue
		}
		result.Document = doc
		result.Found = true
		return result
	}
	sort.SliceStable(result.Warnings, func(i, j int) bool {
		return result.Warnings[i].Command < result.Warnings[j].Command
	})
	return result
}

type commandAttempt struct {
	name string
	args []string
}

func (d Discoverer) attempts(module string) []commandAttempt {
	bus := d.BusCommand
	if bus == "" {
		bus = "bus"
	}
	return []commandAttempt{
		{name: bus, args: []string{module, "help", "--format", "opencli"}},
		{name: "bus-" + module, args: []string{"help", "--format", "opencli"}},
	}
}

func validateModuleName(module string) error {
	if module == "" {
		return fmt.Errorf("module is required")
	}
	if !regexp.MustCompile(`^[a-z][a-z0-9-]*$`).MatchString(module) {
		return fmt.Errorf("invalid module name: %s", module)
	}
	return nil
}

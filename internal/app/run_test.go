package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/busdk/bus-help/pkg/busmeta"
	"github.com/busdk/bus-help/pkg/discovery"
	"github.com/busdk/bus-help/pkg/opencli"
)

func TestRunSelfOpenCLIHelpIncludesEnvironmentMetadata(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"bus-help", "help", "--format", "opencli"}, t.TempDir(), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run self OpenCLI code=%d stderr=%s", code, stderr.String())
	}
	var doc opencli.Document
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("decode OpenCLI: %v\n%s", err, stdout.String())
	}
	if doc.Info.Title != "bus-help" {
		t.Fatalf("unexpected title: %s", doc.Info.Title)
	}
	if _, ok, err := busmeta.EnvironmentFromDocument(doc); err != nil || !ok {
		t.Fatalf("environment metadata ok=%t err=%v", ok, err)
	}
	if !documentOption(doc, "--verbose") || !documentOption(doc, "--quiet") || !documentOption(doc, "--trace") {
		t.Fatalf("OpenCLI document missing diagnostic options: %#v", doc.Commands)
	}
}

func TestRunSelfOpenCLIHelpAllowsDiagnosticFlagsBeforeHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"bus-help", "--quiet", "--format", "opencli", "help"}, t.TempDir(), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run self OpenCLI code=%d stderr=%s", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("self help should not emit diagnostics: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), `"title": "bus-help"`) {
		t.Fatalf("stdout missing self metadata: %s", stdout.String())
	}
}

func TestRunDiscoveryWarningsRespectQuiet(t *testing.T) {
	workdir := t.TempDir()
	withDiscoverer(t, failingDiscoveryRunner{})

	var stdout, stderr bytes.Buffer
	code := Run([]string{"bus-help", "journal"}, workdir, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run code=%d stderr=%s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "bus-help: WARN:") {
		t.Fatalf("default diagnostics should include WARN output, got %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"bus-help", "--quiet", "journal"}, workdir, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run quiet code=%d stderr=%s", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("quiet should suppress non-error diagnostics, got %q", stderr.String())
	}
}

func TestRunVerboseAndTraceDiagnostics(t *testing.T) {
	workdir := t.TempDir()
	withDiscoverer(t, successfulDiscoveryRunner{})

	tests := []struct {
		name      string
		args      []string
		want      string
		forbid    string
		wantTitle string
	}{
		{name: "verbose debug", args: []string{"bus-help", "-v", "journal"}, want: "bus-help: DEBUG:", forbid: "bus-help: TRACE:", wantTitle: "bus-journal"},
		{name: "compact trace", args: []string{"bus-help", "-vv", "journal"}, want: "bus-help: TRACE:", wantTitle: "bus-journal"},
		{name: "trace flag", args: []string{"bus-help", "--trace", "journal"}, want: "bus-help: TRACE:", wantTitle: "bus-journal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tt.args, workdir, &stdout, &stderr)
			if code != 0 {
				t.Fatalf("Run code=%d stderr=%s", code, stderr.String())
			}
			if !strings.Contains(stdout.String(), tt.wantTitle) {
				t.Fatalf("stdout missing title %q: %s", tt.wantTitle, stdout.String())
			}
			if !strings.Contains(stderr.String(), tt.want) {
				t.Fatalf("stderr missing %q: %s", tt.want, stderr.String())
			}
			if tt.forbid != "" && strings.Contains(stderr.String(), tt.forbid) {
				t.Fatalf("stderr should not contain %q: %s", tt.forbid, stderr.String())
			}
		})
	}
}

func TestRunQuietConflictsWithVerboseAndTrace(t *testing.T) {
	tests := [][]string{
		{"bus-help", "--quiet", "--verbose", "journal"},
		{"bus-help", "--quiet", "--trace", "journal"},
	}
	for _, args := range tests {
		var stdout, stderr bytes.Buffer
		code := Run(args, t.TempDir(), &stdout, &stderr)
		if code != 2 {
			t.Fatalf("Run(%v) code=%d want 2", args, code)
		}
		if !strings.Contains(stderr.String(), "mutually exclusive") {
			t.Fatalf("stderr missing conflict diagnostic: %s", stderr.String())
		}
		if stdout.Len() != 0 {
			t.Fatalf("usage error should not write stdout: %s", stdout.String())
		}
	}
}

func documentOption(doc opencli.Document, name string) bool {
	for _, command := range doc.Commands {
		for _, option := range command.Options {
			if option.Name == name {
				return true
			}
		}
	}
	return false
}

func withDiscoverer(t *testing.T, runner discovery.Runner) {
	t.Helper()
	previous := newDiscoverer
	newDiscoverer = func(workdir string) discovery.Discoverer {
		return discovery.Discoverer{
			Runner:     runner,
			BusCommand: "bus",
			Workdir:    workdir,
			UseCache:   false,
		}
	}
	t.Cleanup(func() {
		newDiscoverer = previous
	})
}

type successfulDiscoveryRunner struct{}

func (successfulDiscoveryRunner) Run(ctx context.Context, name string, args []string, dir string) ([]byte, []byte, int, error) {
	if name == "bus" && strings.Join(args, "\x00") == "journal\x00help\x00--format\x00opencli" {
		return []byte(`{"opencli":"0.1.0","info":{"title":"bus-journal","summary":"Journal help"}}`), nil, 0, nil
	}
	return nil, []byte("unexpected discovery command"), 2, errors.New("unexpected discovery command")
}

type failingDiscoveryRunner struct{}

func (failingDiscoveryRunner) Run(ctx context.Context, name string, args []string, dir string) ([]byte, []byte, int, error) {
	return nil, []byte("metadata unavailable"), 1, errors.New("metadata unavailable")
}

package discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakeRunner struct {
	calls int
}

func (f *fakeRunner) Run(ctx context.Context, name string, args []string, dir string) ([]byte, []byte, int, error) {
	f.calls++
	if f.calls == 1 {
		return nil, []byte("missing"), 127, errors.New("not found")
	}
	return []byte(`{"opencli":"0.1.0","info":{"title":"bus-journal"}}`), nil, 0, nil
}

func TestDiscoverModuleFallsBackAndParsesOpenCLI(t *testing.T) {
	runner := &fakeRunner{}
	result := Discoverer{Runner: runner, BusCommand: "bus"}.DiscoverModule(context.Background(), "journal")
	if !result.Found {
		t.Fatalf("expected document, warnings=%v", result.Warnings)
	}
	if result.Document.Info.Title != "bus-journal" {
		t.Fatalf("title = %q", result.Document.Info.Title)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("warnings = %d, want 1", len(result.Warnings))
	}
}

func TestDiscoverModuleFallsBackToSuperprojectBinary(t *testing.T) {
	workdir := t.TempDir()
	binaryPath := filepath.Join(workdir, "bus-journal", "bin", "bus-journal")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &pathSensitiveRunner{wantName: binaryPath}
	result := Discoverer{Runner: runner, BusCommand: "missing-bus", Workdir: workdir}.DiscoverModule(context.Background(), "journal")
	if !result.Found {
		t.Fatalf("expected document from local binary, warnings=%v", result.Warnings)
	}
	if !runner.sawLocalBinary {
		t.Fatalf("runner never saw local binary; calls=%v", runner.calls)
	}
}

type pathSensitiveRunner struct {
	wantName       string
	sawLocalBinary bool
	calls          []string
}

func (r *pathSensitiveRunner) Run(ctx context.Context, name string, args []string, dir string) ([]byte, []byte, int, error) {
	r.calls = append(r.calls, name)
	if name != r.wantName {
		return nil, []byte("missing"), 127, errors.New("not found")
	}
	r.sawLocalBinary = true
	return []byte(`{"opencli":"0.1.0","info":{"title":"bus-journal"}}`), nil, 0, nil
}

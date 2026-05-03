package discovery

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
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

func TestDiscoverModuleUsesValidatedStdoutCache(t *testing.T) {
	workdir := t.TempDir()
	binaryPath := filepath.Join(workdir, "bus-journal", "bin", "bus-journal")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &pathSensitiveRunner{wantName: binaryPath}
	d := Discoverer{Runner: runner, BusCommand: "missing-bus", Workdir: workdir, CacheDir: filepath.Join(workdir, "cache"), UseCache: true}
	first := d.DiscoverModule(context.Background(), "journal")
	if !first.Found {
		t.Fatalf("first discovery failed: %#v", first.Warnings)
	}
	second := d.DiscoverModule(context.Background(), "journal")
	if !second.Found {
		t.Fatalf("second discovery failed: %#v", second.Warnings)
	}
	if runner.sawLocalBinaryCount != 1 {
		t.Fatalf("runner calls to local binary = %d, want 1", runner.sawLocalBinaryCount)
	}
}

func TestDiscoverModuleWarmCacheAvoidsRunnerAcrossDiscoverers(t *testing.T) {
	workdir := t.TempDir()
	cacheDir := filepath.Join(workdir, "cache")
	binaryPath := filepath.Join(workdir, "bus-journal", "bin", "bus-journal")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	firstRunner := &pathSensitiveRunner{wantName: binaryPath}
	first := Discoverer{Runner: firstRunner, BusCommand: "missing-bus", Workdir: workdir, CacheDir: cacheDir, UseCache: true}.DiscoverModule(context.Background(), "journal")
	if !first.Found {
		t.Fatalf("first discovery failed: %#v", first.Warnings)
	}
	failingRunner := &pathSensitiveRunner{wantName: filepath.Join(workdir, "missing")}
	second := Discoverer{Runner: failingRunner, BusCommand: "missing-bus", Workdir: workdir, CacheDir: cacheDir, UseCache: true}.DiscoverModule(context.Background(), "journal")
	if !second.Found {
		t.Fatalf("second discovery should be served from cache: %#v", second.Warnings)
	}
	if failingRunner.sawLocalBinaryCount != 0 {
		t.Fatalf("warm cache executed runner %d times", failingRunner.sawLocalBinaryCount)
	}
}

func TestDiscoverModuleCacheDirEnvOverride(t *testing.T) {
	workdir := t.TempDir()
	cacheDir := filepath.Join(workdir, "env-cache")
	t.Setenv("BUS_OPENCLI_DISCOVERY_CACHE_DIR", cacheDir)
	binaryPath := filepath.Join(workdir, "bus-journal", "bin", "bus-journal")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &pathSensitiveRunner{wantName: binaryPath}
	result := Discoverer{Runner: runner, BusCommand: "missing-bus", Workdir: workdir, UseCache: true}.DiscoverModule(context.Background(), "journal")
	if !result.Found {
		t.Fatalf("discovery failed: %#v", result.Warnings)
	}
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		t.Fatalf("read cache dir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("cache entries = %d, want 1", len(entries))
	}
}

func TestDiscoverModuleCachesFailureForUnchangedLocalBinary(t *testing.T) {
	workdir := t.TempDir()
	cacheDir := filepath.Join(workdir, "cache")
	binaryPath := filepath.Join(workdir, "bus-shell", "bin", "bus-shell")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &failingPathRunner{wantName: binaryPath}
	d := Discoverer{Runner: runner, BusCommand: "missing-bus", Workdir: workdir, CacheDir: cacheDir, UseCache: true}
	first := d.DiscoverModule(context.Background(), "shell")
	if first.Found || len(first.Warnings) != 1 {
		t.Fatalf("first discovery found=%t warnings=%#v", first.Found, first.Warnings)
	}
	second := d.DiscoverModule(context.Background(), "shell")
	if second.Found || len(second.Warnings) != 1 {
		t.Fatalf("second discovery found=%t warnings=%#v", second.Found, second.Warnings)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
}

func TestDiscoverModuleDoesNotCacheTimeoutForUnchangedLocalBinary(t *testing.T) {
	workdir := t.TempDir()
	cacheDir := filepath.Join(workdir, "cache")
	binaryPath := filepath.Join(workdir, "bus-shell", "bin", "bus-shell")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &timeoutPathRunner{wantName: binaryPath}
	d := Discoverer{Runner: runner, BusCommand: "missing-bus", Workdir: workdir, CacheDir: cacheDir, Timeout: time.Millisecond, UseCache: true}
	first := d.DiscoverModule(context.Background(), "shell")
	if first.Found || len(first.Warnings) != 1 || first.Warnings[0].Message != "metadata command timed out" {
		t.Fatalf("first discovery found=%t warnings=%#v", first.Found, first.Warnings)
	}
	second := d.DiscoverModule(context.Background(), "shell")
	if second.Found || len(second.Warnings) != 1 || second.Warnings[0].Message != "metadata command timed out" {
		t.Fatalf("second discovery found=%t warnings=%#v", second.Found, second.Warnings)
	}
	if runner.calls != 2 {
		t.Fatalf("runner calls = %d, want timeout to be retried", runner.calls)
	}
}

func TestDiscoverModuleStopsAfterLocalBinaryFailure(t *testing.T) {
	workdir := t.TempDir()
	binaryPath := filepath.Join(workdir, "bus-faq", "bin", "bus-faq")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &failingPathRunner{wantName: binaryPath}
	result := Discoverer{Runner: runner, BusCommand: "missing-bus", Workdir: workdir, UseCache: false}.DiscoverModule(context.Background(), "faq")
	if result.Found || len(result.Warnings) != 1 {
		t.Fatalf("found=%t warnings=%#v", result.Found, result.Warnings)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want only local attempt", runner.calls)
	}
}

func BenchmarkDiscoverModuleColdFakeRunner(b *testing.B) {
	workdir := b.TempDir()
	binaryPath := filepath.Join(workdir, "bus-journal", "bin", "bus-journal")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		runner := &pathSensitiveRunner{wantName: binaryPath}
		d := Discoverer{Runner: runner, BusCommand: "missing-bus", Workdir: workdir, CacheDir: filepath.Join(workdir, "cache", strconv.Itoa(i)), UseCache: false}
		result := d.DiscoverModule(context.Background(), "journal")
		if !result.Found {
			b.Fatalf("discovery failed: %#v", result.Warnings)
		}
	}
}

func BenchmarkDiscoverModuleWarmStdoutCache(b *testing.B) {
	workdir := b.TempDir()
	cacheDir := filepath.Join(workdir, "cache")
	binaryPath := filepath.Join(workdir, "bus-journal", "bin", "bus-journal")
	if err := os.MkdirAll(filepath.Dir(binaryPath), 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\n"), 0o755); err != nil {
		b.Fatal(err)
	}
	runner := &pathSensitiveRunner{wantName: binaryPath}
	d := Discoverer{Runner: runner, BusCommand: "missing-bus", Workdir: workdir, CacheDir: cacheDir, UseCache: true}
	result := d.DiscoverModule(context.Background(), "journal")
	if !result.Found {
		b.Fatalf("initial discovery failed: %#v", result.Warnings)
	}
	failingRunner := &pathSensitiveRunner{wantName: filepath.Join(workdir, "missing")}
	warm := Discoverer{Runner: failingRunner, BusCommand: "missing-bus", Workdir: workdir, CacheDir: cacheDir, UseCache: true}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result := warm.DiscoverModule(context.Background(), "journal")
		if !result.Found {
			b.Fatalf("warm discovery failed: %#v", result.Warnings)
		}
	}
	b.StopTimer()
	if failingRunner.sawLocalBinaryCount != 0 {
		b.Fatalf("warm cache executed runner %d times", failingRunner.sawLocalBinaryCount)
	}
}

type pathSensitiveRunner struct {
	wantName            string
	sawLocalBinary      bool
	sawLocalBinaryCount int
	calls               []string
}

func (r *pathSensitiveRunner) Run(ctx context.Context, name string, args []string, dir string) ([]byte, []byte, int, error) {
	r.calls = append(r.calls, name)
	if name != r.wantName {
		return nil, []byte("missing"), 127, errors.New("not found")
	}
	r.sawLocalBinary = true
	r.sawLocalBinaryCount++
	return []byte(`{"opencli":"0.1.0","info":{"title":"bus-journal"}}`), nil, 0, nil
}

type failingPathRunner struct {
	wantName string
	calls    int
}

func (r *failingPathRunner) Run(ctx context.Context, name string, args []string, dir string) ([]byte, []byte, int, error) {
	if name != r.wantName {
		return nil, []byte("unexpected fallback"), 127, errors.New("unexpected fallback")
	}
	r.calls++
	return nil, []byte("unsupported metadata"), 2, errors.New("unsupported metadata")
}

type timeoutPathRunner struct {
	wantName string
	calls    int
}

func (r *timeoutPathRunner) Run(ctx context.Context, name string, args []string, dir string) ([]byte, []byte, int, error) {
	if name != r.wantName {
		return nil, []byte("unexpected fallback"), 127, errors.New("unexpected fallback")
	}
	r.calls++
	<-ctx.Done()
	return nil, nil, 1, ctx.Err()
}

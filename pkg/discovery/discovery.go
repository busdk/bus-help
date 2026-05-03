// Package discovery finds module-owned live metadata by executing Bus module help
// commands with explicit argv arrays.
package discovery

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	CacheDir   string
	UseCache   bool
}

// New creates a production Discoverer.
//
// Used by: CLI app constructors.
func New(workdir string) Discoverer {
	return Discoverer{
		Runner:     ExecRunner{},
		BusCommand: "bus",
		Timeout:    750 * time.Millisecond,
		Workdir:    workdir,
		UseCache:   true,
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
		timeout = 750 * time.Millisecond
	}
	attempts := d.attempts(module)
	for _, attempt := range attempts {
		commandText := strings.TrimSpace(attempt.name + " " + strings.Join(attempt.args, " "))
		if d.cacheEnabled() {
			if cached, ok := d.readCachedAttempt(attempt); ok {
				if !cached.Success {
					if cached.Message == "metadata command timed out" {
						continue
					}
					result.Warnings = append(result.Warnings, Warning{Module: module, Command: commandText, Message: cached.Message})
					if attempt.local {
						break
					}
					continue
				}
				var doc opencli.Document
				if err := json.Unmarshal(cached.Stdout, &doc); err == nil {
					result.Document = doc
					result.Found = true
					return result
				}
			}
		}
		attemptCtx, cancel := context.WithTimeout(ctx, timeout)
		stdout, stderr, exitCode, err := runner.Run(attemptCtx, attempt.name, attempt.args, d.Workdir)
		cancel()
		if attemptCtx.Err() == context.DeadlineExceeded {
			result.Warnings = append(result.Warnings, Warning{Module: module, Command: commandText, Message: "metadata command timed out"})
			if attempt.local {
				break
			}
			continue
		}
		if err != nil || exitCode != 0 {
			msg := warningMessage(stderr, err)
			result.Warnings = append(result.Warnings, Warning{Module: module, Command: commandText, Message: msg})
			if d.cacheEnabled() {
				d.writeCachedFailure(attempt, exitCode, stderr, msg)
			}
			if attempt.local {
				break
			}
			continue
		}
		var doc opencli.Document
		if err := json.Unmarshal(stdout, &doc); err != nil {
			msg := fmt.Sprintf("invalid OpenCLI JSON: %s", err)
			result.Warnings = append(result.Warnings, Warning{Module: module, Command: commandText, Message: msg})
			if d.cacheEnabled() {
				d.writeCachedFailure(attempt, exitCode, stderr, msg)
			}
			if attempt.local {
				break
			}
			continue
		}
		if d.cacheEnabled() {
			d.writeCachedSuccess(attempt, stdout)
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
	name  string
	args  []string
	local bool
}

type discoveryCacheEntry struct {
	CacheVersion string   `json:"cacheVersion"`
	ResolvedPath string   `json:"resolvedPath"`
	Args         []string `json:"args"`
	Size         int64    `json:"size"`
	ModTimeUnix  int64    `json:"modTimeUnix"`
	Success      bool     `json:"success"`
	Stdout       []byte   `json:"stdout"`
	Stderr       []byte   `json:"stderr,omitempty"`
	ExitCode     int      `json:"exitCode,omitempty"`
	Message      string   `json:"message,omitempty"`
}

// discoveryCacheVersion invalidates old OpenCLI stdout cache entries.
//
// Used by: readCachedAttempt, writeCachedEntry, and cacheMetadataAll.
const discoveryCacheVersion = "2"

func (d Discoverer) attempts(module string) []commandAttempt {
	bus := d.BusCommand
	if bus == "" {
		bus = "bus"
	}
	var attempts []commandAttempt
	for _, path := range d.superprojectBinaryCandidates(module) {
		attempts = append(attempts, commandAttempt{name: path, args: []string{"help", "--format", "opencli"}, local: true})
	}
	attempts = append(attempts,
		commandAttempt{name: bus, args: []string{module, "help", "--format", "opencli"}},
		commandAttempt{name: "bus-" + module, args: []string{"help", "--format", "opencli"}},
	)
	return attempts
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

// warningMessage returns a concise diagnostic for a failed metadata command.
//
// Used by: DiscoverModule.
func warningMessage(stderr []byte, err error) string {
	msg := strings.TrimSpace(string(stderr))
	if msg == "" && err != nil {
		msg = err.Error()
	}
	return msg
}

// cacheEnabled reports whether live OpenCLI stdout caching may be used.
//
// Used by: DiscoverModule.
func (d Discoverer) cacheEnabled() bool {
	return d.UseCache && os.Getenv("BUS_OPENCLI_DISCOVERY_CACHE") != "0"
}

// readCachedAttempt returns a cached success or failure when the binary is unchanged.
//
// Used by: DiscoverModule.
func (d Discoverer) readCachedAttempt(attempt commandAttempt) (discoveryCacheEntry, bool) {
	for _, meta := range d.cacheMetadataAll(attempt) {
		data, err := os.ReadFile(meta.cachePath)
		if err != nil {
			continue
		}
		var entry discoveryCacheEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue
		}
		if entry.CacheVersion != discoveryCacheVersion || entry.ResolvedPath != meta.resolvedPath || entry.Size != meta.size || entry.ModTimeUnix != meta.modTimeUnix || strings.Join(entry.Args, "\x00") != strings.Join(attempt.args, "\x00") {
			continue
		}
		if !entry.Success && entry.Message == "" {
			continue
		}
		if entry.Success && len(entry.Stdout) == 0 {
			continue
		}
		return entry, true
	}
	return discoveryCacheEntry{}, false
}

// writeCachedSuccess stores successful live OpenCLI stdout for unchanged binaries.
//
// Used by: DiscoverModule.
func (d Discoverer) writeCachedSuccess(attempt commandAttempt, stdout []byte) {
	d.writeCachedEntry(attempt, discoveryCacheEntry{Success: true, Stdout: append([]byte(nil), stdout...)})
}

// writeCachedFailure stores non-success metadata discovery outcomes.
//
// Used by: DiscoverModule.
func (d Discoverer) writeCachedFailure(attempt commandAttempt, exitCode int, stderr []byte, message string) {
	d.writeCachedEntry(attempt, discoveryCacheEntry{Success: false, Stderr: append([]byte(nil), stderr...), ExitCode: exitCode, Message: message})
}

// writeCachedEntry stores one validated discovery cache entry.
//
// Used by: writeCachedSuccess and writeCachedFailure.
func (d Discoverer) writeCachedEntry(attempt commandAttempt, entry discoveryCacheEntry) {
	metas := d.cacheMetadataAll(attempt)
	if len(metas) == 0 {
		return
	}
	for _, meta := range metas {
		if err := os.MkdirAll(filepath.Dir(meta.cachePath), 0o700); err != nil {
			continue
		}
		next := entry
		next.CacheVersion = discoveryCacheVersion
		next.ResolvedPath = meta.resolvedPath
		next.Args = append([]string(nil), attempt.args...)
		next.Size = meta.size
		next.ModTimeUnix = meta.modTimeUnix
		data, err := json.Marshal(next)
		if err != nil {
			return
		}
		if err := os.WriteFile(meta.cachePath, data, 0o600); err == nil {
			return
		}
	}
}

type cacheMetadata struct {
	resolvedPath string
	size         int64
	modTimeUnix  int64
	cachePath    string
}

// cacheMetadata returns the cache key material for one command attempt.
//
// Used by: readCachedStdout and writeCachedStdout.
func (d Discoverer) cacheMetadata(attempt commandAttempt) (cacheMetadata, bool) {
	metas := d.cacheMetadataAll(attempt)
	if len(metas) == 0 {
		return cacheMetadata{}, false
	}
	return metas[0], true
}

// cacheMetadataAll returns cache key material for each eligible cache directory.
//
// Used by: readCachedStdout, writeCachedStdout, and cacheMetadata.
func (d Discoverer) cacheMetadataAll(attempt commandAttempt) []cacheMetadata {
	resolved, ok := resolveCommandPath(attempt.name)
	if !ok || filepath.Base(resolved) == filepath.Base(d.BusCommand) && !strings.Contains(attempt.name, string(filepath.Separator)) {
		return nil
	}
	info, err := os.Stat(resolved)
	if err != nil || info.IsDir() {
		return nil
	}
	keyData := strings.Join(append([]string{discoveryCacheVersion, resolved}, attempt.args...), "\x00")
	sum := sha256.Sum256([]byte(keyData))
	cacheName := hex.EncodeToString(sum[:]) + ".json"
	var metas []cacheMetadata
	for _, cacheDir := range d.cacheDirs() {
		metas = append(metas, cacheMetadata{
			resolvedPath: resolved,
			size:         info.Size(),
			modTimeUnix:  info.ModTime().UnixNano(),
			cachePath:    filepath.Join(cacheDir, cacheName),
		})
	}
	return metas
}

// cacheDir returns the directory used for validated discovery stdout cache files.
//
// Used by: cacheMetadata.
func (d Discoverer) cacheDir() string {
	dirs := d.cacheDirs()
	if len(dirs) == 0 {
		return ""
	}
	return dirs[0]
}

// cacheDirs returns candidate directories for validated discovery stdout cache files.
//
// Used by: cacheMetadataAll and cacheDir.
func (d Discoverer) cacheDirs() []string {
	if envDir := strings.TrimSpace(os.Getenv("BUS_OPENCLI_DISCOVERY_CACHE_DIR")); envDir != "" {
		return []string{envDir}
	}
	if d.CacheDir != "" {
		return []string{d.CacheDir}
	}
	var dirs []string
	dir, err := os.UserCacheDir()
	if err == nil && dir != "" {
		dirs = append(dirs, filepath.Join(dir, "busdk", "opencli-discovery"))
	}
	dirs = append(dirs, filepath.Join(os.TempDir(), "busdk", "opencli-discovery"))
	return sortedUniquePaths(dirs)
}

// resolveCommandPath resolves command names without executing a shell.
//
// Used by: cacheMetadata.
func resolveCommandPath(name string) (string, bool) {
	if strings.Contains(name, string(filepath.Separator)) {
		abs, err := filepath.Abs(name)
		if err != nil {
			return "", false
		}
		return abs, true
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", false
	}
	return path, true
}

// sortedUniquePaths returns deterministic cleaned path values.
//
// Used by: cacheDirs.
func sortedUniquePaths(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		clean := filepath.Clean(value)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		out = append(out, clean)
	}
	return out
}

// superprojectBinaryCandidates returns existing module-local binaries near Workdir.
//
// Used by: attempts for BusDK superproject checkout discovery.
func (d Discoverer) superprojectBinaryCandidates(module string) []string {
	if d.Workdir == "" {
		return nil
	}
	moduleDir := "bus-" + module
	binary := moduleDir
	candidates := []string{
		filepath.Join(d.Workdir, moduleDir, "bin", binary),
		filepath.Join(filepath.Dir(d.Workdir), moduleDir, "bin", binary),
	}
	var out []string
	seen := map[string]bool{}
	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		if seen[clean] {
			continue
		}
		seen[clean] = true
		info, err := os.Stat(clean)
		if err != nil || info.IsDir() {
			continue
		}
		out = append(out, clean)
	}
	sort.Strings(out)
	return out
}

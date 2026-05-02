package discovery

import (
	"context"
	"errors"
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

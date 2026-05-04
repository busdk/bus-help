package diagnostics

import (
	"flag"
	"testing"
)

func TestLevelFor(t *testing.T) {
	tests := []struct {
		name    string
		quiet   bool
		trace   bool
		verbose int
		want    Level
		wantErr bool
	}{
		{name: "default info", want: LevelInfo},
		{name: "quiet error", quiet: true, want: LevelError},
		{name: "single verbose debug", verbose: 1, want: LevelDebug},
		{name: "double verbose trace", verbose: 2, want: LevelTrace},
		{name: "trace flag trace", trace: true, want: LevelTrace},
		{name: "quiet conflicts with verbose", quiet: true, verbose: 1, want: LevelError, wantErr: true},
		{name: "quiet conflicts with trace", quiet: true, trace: true, want: LevelError, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LevelFor(tt.quiet, tt.trace, tt.verbose)
			if (err != nil) != tt.wantErr {
				t.Fatalf("LevelFor() err=%v wantErr=%v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("LevelFor()=%s want %s", got, tt.want)
			}
		})
	}
}

func TestFlagsParseVerboseForms(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Level
	}{
		{name: "short", args: []string{"-v"}, want: LevelDebug},
		{name: "long", args: []string{"--verbose"}, want: LevelDebug},
		{name: "compact", args: []string{"-vv"}, want: LevelTrace},
		{name: "repeated", args: []string{"--verbose", "--verbose"}, want: LevelTrace},
		{name: "trace alias", args: []string{"--trace"}, want: LevelTrace},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diagnosticFlags Flags
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			AddFlags(fs, &diagnosticFlags)
			if err := fs.Parse(ExpandVerbosityArgs(tt.args)); err != nil {
				t.Fatalf("Parse() err=%v", err)
			}
			got, err := diagnosticFlags.Level()
			if err != nil {
				t.Fatalf("Level() err=%v", err)
			}
			if got != tt.want {
				t.Fatalf("Level()=%s want %s", got, tt.want)
			}
		})
	}
}

func TestExpandVerbosityArgsPreservesOtherArgs(t *testing.T) {
	got := ExpandVerbosityArgs([]string{"-vv", "--", "-vv", "-abc"})
	want := []string{"-v", "-v", "--", "-vv", "-abc"}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("arg[%d]=%q want %q: %#v", i, got[i], want[i], got)
		}
	}
}

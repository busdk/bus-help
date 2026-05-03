package modulehelp

import (
	"bytes"
	"strings"
	"testing"

	"github.com/busdk/bus-help/pkg/busmeta"
	"github.com/busdk/bus-help/pkg/opencli"
)

func TestHandleOpenCLI(t *testing.T) {
	var stdout bytes.Buffer
	handled, code := Handle([]string{"help", "--format", "opencli"}, &stdout, ioDiscard{}, "text help\n", opencli.Document{
		OpenCLI: "0.1.0",
		Info:    opencli.Info{Title: "bus-test"},
	})
	if !handled || code != 0 {
		t.Fatalf("handled=%v code=%d", handled, code)
	}
	if !strings.Contains(stdout.String(), `"opencli": "0.1.0"`) {
		t.Fatalf("stdout missing OpenCLI document: %s", stdout.String())
	}
}

func TestHandleText(t *testing.T) {
	var stdout bytes.Buffer
	handled, code := Handle([]string{"help"}, &stdout, ioDiscard{}, "text help\n", opencli.Document{})
	if !handled || code != 0 {
		t.Fatalf("handled=%v code=%d", handled, code)
	}
	if stdout.String() != "text help\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestSimpleDocumentIncludesEnvironmentMetadata(t *testing.T) {
	doc := SimpleDocument("example", "bus-example", "Example command.", []EnvVar{
		{Name: "BUS_EXAMPLE_TOKEN", Description: "Example token.", Secret: true},
	})
	env, ok, err := busmeta.EnvironmentFromDocument(doc)
	if err != nil {
		t.Fatalf("environment metadata invalid: %v", err)
	}
	if !ok || len(env.Variables) != 1 {
		t.Fatalf("environment metadata missing: ok=%t vars=%#v", ok, env.Variables)
	}
	if env.Variables[0].Name != "BUS_EXAMPLE_TOKEN" || !env.Variables[0].Secret {
		t.Fatalf("unexpected variable: %#v", env.Variables[0])
	}
}

func TestSimpleDocumentReplacesPlaceholderEnvironmentDescription(t *testing.T) {
	doc := SimpleDocument("example", "bus-example", "Example command.", []EnvVar{
		{Name: "BUS_E2E_VERBOSE", Description: "BUS_E2E_VERBOSE setting used by this Bus module."},
		{Name: "BUS_UNKNOWN_THING", Description: "BUS_UNKNOWN_THING setting used by this Bus module."},
	})
	env, ok, err := busmeta.EnvironmentFromDocument(doc)
	if err != nil || !ok {
		t.Fatalf("environment metadata ok=%t err=%v", ok, err)
	}
	if env.Variables[0].Description == "" || strings.Contains(env.Variables[0].Description, "setting used by this Bus module") {
		t.Fatalf("standard description not applied: %#v", env.Variables[0])
	}
	if env.Variables[1].Description != "" {
		t.Fatalf("unknown placeholder should not be emitted as real description: %#v", env.Variables[1])
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

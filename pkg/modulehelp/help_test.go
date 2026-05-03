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

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

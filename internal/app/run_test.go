package app

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/busdk/bus-help/pkg/busmeta"
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
}

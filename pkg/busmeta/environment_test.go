package busmeta

import (
	"encoding/json"
	"testing"

	"github.com/busdk/bus-help/pkg/opencli"
)

func TestEnvironmentRoundTripAndValueValidation(t *testing.T) {
	doc := opencli.Document{OpenCLI: "0.1.0", Info: opencli.Info{Title: "bus-test"}}
	AttachEnvironment(&doc, "test", EnvironmentMetadata{
		Version: "0.1",
		Variables: []EnvironmentVariable{
			{Name: "BUS_TEST_MODE", Required: true, Schema: Schema{Type: "string", Enum: []string{"on", "off"}}},
		},
	})
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	var decoded opencli.Document
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	env, ok, err := EnvironmentFromDocument(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected environment metadata")
	}
	if err := ValidateValue(env.Variables[0], "maybe"); err == nil {
		t.Fatal("expected enum validation failure")
	}
	if err := ValidateValue(env.Variables[0], "on"); err != nil {
		t.Fatal(err)
	}
}

func TestRedactedValue(t *testing.T) {
	variable := EnvironmentVariable{Name: "BUS_SECRET", Secret: true}
	if got := RedactedValue(variable, "secret-value"); got != "********" {
		t.Fatalf("RedactedValue() = %q", got)
	}
}

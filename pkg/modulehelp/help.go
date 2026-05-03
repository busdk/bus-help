// Package modulehelp provides a small shared adapter for Bus module binaries
// that expose live OpenCLI-compatible help metadata.
package modulehelp

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/busdk/bus-help/pkg/opencli"
)

// Handle writes module-local help output when args request the help subcommand.
//
// Used by: Bus module binaries before their normal flag parsing.
func Handle(args []string, stdout io.Writer, stderr io.Writer, textHelp string, doc opencli.Document) (bool, int) {
	if len(args) == 0 || args[0] != "help" {
		return false, 0
	}
	format, err := parseFormat(args[1:])
	if err != nil {
		fmt.Fprintf(stderr, "help: %s\n", err)
		return true, 2
	}
	switch format {
	case "", "text":
		_, _ = io.WriteString(stdout, textHelp)
		return true, 0
	case "opencli", "json":
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(doc); err != nil {
			fmt.Fprintf(stderr, "help: %s\n", err)
			return true, 1
		}
		return true, 0
	default:
		fmt.Fprintf(stderr, "help: unsupported format: %s\n", format)
		return true, 2
	}
}

func parseFormat(args []string) (string, error) {
	format := "text"
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return "text", nil
		case arg == "-f" || arg == "--format":
			if i+1 >= len(args) {
				return "", fmt.Errorf("missing value for %s", arg)
			}
			i++
			format = args[i]
		case strings.HasPrefix(arg, "--format="):
			format = strings.TrimPrefix(arg, "--format=")
		default:
			return "", fmt.Errorf("unknown help argument: %s", arg)
		}
	}
	return format, nil
}

// Package opencli contains the small OpenCLI 0.1-compatible model Bus needs for
// live command metadata.
package opencli

// Document is the root OpenCLI-compatible help document.
//
// Used by: bus-help rendering, module-owned metadata output, and bus-configure discovery.
type Document struct {
	OpenCLI  string         `json:"opencli"`
	Info     Info           `json:"info"`
	Commands []Command      `json:"commands,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// Info describes the CLI or module represented by a Document.
//
// Used by: Document.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
}

// Command describes one command path in an OpenCLI-compatible document.
//
// Used by: Document and nested command metadata.
type Command struct {
	Name        string         `json:"name"`
	Summary     string         `json:"summary,omitempty"`
	Description string         `json:"description,omitempty"`
	Usage       string         `json:"usage,omitempty"`
	Arguments   []Argument     `json:"arguments,omitempty"`
	Options     []Option       `json:"options,omitempty"`
	Commands    []Command      `json:"commands,omitempty"`
	Examples    []Example      `json:"examples,omitempty"`
	ExitCodes   []ExitCode     `json:"exitCodes,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// Argument describes a positional command argument.
//
// Used by: Command.
type Argument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Repeatable  bool   `json:"repeatable,omitempty"`
}

// Option describes a command option or flag.
//
// Used by: Command.
type Option struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases,omitempty"`
	Description string   `json:"description,omitempty"`
	ValueName   string   `json:"valueName,omitempty"`
	Required    bool     `json:"required,omitempty"`
	Repeatable  bool     `json:"repeatable,omitempty"`
	Default     string   `json:"default,omitempty"`
}

// Example describes one deterministic command example.
//
// Used by: Command.
type Example struct {
	Summary string `json:"summary,omitempty"`
	Command string `json:"command"`
}

// ExitCode describes one possible command exit code.
//
// Used by: Command.
type ExitCode struct {
	Code        int    `json:"code"`
	Description string `json:"description,omitempty"`
}

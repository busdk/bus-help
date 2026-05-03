package modulehelp

import (
	"strings"

	"github.com/busdk/bus-help/pkg/busmeta"
	"github.com/busdk/bus-help/pkg/opencli"
)

// EnvVar describes a module-owned environment variable for simple OpenCLI metadata.
//
// Used by: Bus module binaries that do not need custom OpenCLI construction.
type EnvVar struct {
	Name          string
	Description   string
	Required      bool
	Secret        bool
	Default       string
	Enum          []string
	Pattern       string
	Format        string
	Affects       []string
	Scope         string
	StoreInDotenv bool
}

// SimpleDocument returns a deterministic OpenCLI document for a Bus module.
//
// Used by: module-owned metadata adapters in Bus command binaries.
func SimpleDocument(module string, binary string, summary string, variables []EnvVar) opencli.Document {
	if binary == "" {
		binary = "bus-" + module
	}
	doc := opencli.Document{
		OpenCLI: "0.1.0",
		Info: opencli.Info{
			Title:       binary,
			Version:     "dev",
			Summary:     summary,
			Description: summary,
		},
		Commands: []opencli.Command{{
			Name:    "command",
			Summary: summary,
			Usage:   binary + " [options] [command...]",
			Options: []opencli.Option{
				{Name: "--help", Aliases: []string{"-h"}, Description: "Show human-readable help and exit."},
				{Name: "--version", Aliases: []string{"-V"}, Description: "Print version information and exit."},
				{Name: "--format", Aliases: []string{"-f"}, ValueName: "format", Description: "Select output format where supported."},
			},
			Examples: []opencli.Example{
				{Summary: "Show human help.", Command: binary + " --help"},
				{Summary: "Show machine-readable metadata.", Command: binary + " help --format opencli"},
			},
			ExitCodes: []opencli.ExitCode{
				{Code: 0, Description: "Success."},
				{Code: 1, Description: "Runtime error."},
				{Code: 2, Description: "Usage error."},
			},
		}},
	}
	env := busmeta.EnvironmentMetadata{
		Version:    "0.1",
		Precedence: []string{"CLI flags", ".env", "process environment", "module defaults"},
		Dotenv:     []busmeta.DotenvHint{{Path: ".env", Description: "Workspace or deployment-local Bus environment."}},
		Variables:  make([]busmeta.EnvironmentVariable, 0, len(variables)),
	}
	for _, variable := range variables {
		if converted, ok := variable.toBusMeta(); ok {
			env.Variables = append(env.Variables, converted)
		}
	}
	busmeta.AttachEnvironment(&doc, module, env)
	return doc
}

// SimpleTextHelp returns concise text for the module-local help subcommand.
//
// Used by: module-owned metadata adapters in Bus command binaries.
func SimpleTextHelp(binary string, summary string) string {
	return binary + " exposes live Bus help metadata.\n\nUsage:\n  " + binary + " help [--format text|opencli|json]\n\n" + summary + "\n"
}

func (v EnvVar) toBusMeta() (busmeta.EnvironmentVariable, bool) {
	name := strings.TrimSpace(v.Name)
	if !validEnvName(name) {
		return busmeta.EnvironmentVariable{}, false
	}
	safe := busmeta.SafeHandling{
		Printable:     !v.Secret,
		StoreInDotenv: true,
		RedactInLogs:  v.Secret,
	}
	if !v.StoreInDotenv && v.StoreInDotenvSet() {
		safe.StoreInDotenv = false
	}
	schema := busmeta.Schema{Type: "string", Default: v.Default, Format: v.Format, Enum: v.Enum, Pattern: v.Pattern}
	scope := v.Scope
	if scope == "" {
		scope = "deployment"
	}
	affects := v.Affects
	if len(affects) == 0 {
		affects = []string{inferAffects(name)}
	}
	return busmeta.EnvironmentVariable{
		Name:         name,
		Description:  v.description(),
		Required:     v.Required,
		Secret:       v.Secret,
		Default:      v.Default,
		Schema:       schema,
		SafeHandling: safe,
		Affects:      affects,
		Scope:        scope,
	}, true
}

// validEnvName reports whether a discovered token is a real environment variable.
//
// Used by: EnvVar OpenCLI conversion.
func validEnvName(name string) bool {
	if name == "" || strings.HasSuffix(name, "_") {
		return false
	}
	for i, r := range name {
		if r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' && i > 0 || r == '_' {
			continue
		}
		return false
	}
	return true
}

// description returns module-provided or standard purpose text for an env var.
//
// Used by: EnvVar OpenCLI conversion.
func (v EnvVar) description() string {
	description := strings.TrimSpace(v.Description)
	if description == "" || description == v.Name+" setting used by this Bus module." {
		if standard := standardDescription(v.Name); standard != "" {
			return standard
		}
		return ""
	}
	return description
}

// StoreInDotenvSet reports whether StoreInDotenv was intentionally set false.
//
// Used by: EnvVar conversion.
func (v EnvVar) StoreInDotenvSet() bool {
	return v.Secret && strings.Contains(v.Description, "must not be stored")
}

func inferAffects(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "key") || strings.Contains(lower, "password"):
		return "security"
	case strings.Contains(lower, "database") || strings.Contains(lower, "postgres") || strings.Contains(lower, "dsn"):
		return "storage"
	case strings.Contains(lower, "api") || strings.Contains(lower, "url"):
		return "api"
	default:
		return "configuration"
	}
}

// standardDescription returns shared purpose text for common Bus environment variables.
//
// Used by: EnvVar.description.
func standardDescription(name string) string {
	if description, ok := knownDescription(name); ok {
		return description
	}
	switch {
	case name == "BUS_DEV":
		return "Path or command used to invoke the local bus-dev helper during module development checks."
	case name == "BUS_E2E_KEEP":
		return "Keep temporary e2e workspaces after a test run for debugging."
	case name == "BUS_E2E_VERBOSE":
		return "Enable verbose e2e script logging for troubleshooting."
	case name == "BUS_GO_QUALITY_PROFILE":
		return "bus-dev quality lint profile used by this module's Makefile lint target."
	case name == "BUS_API_TOKEN":
		return "Bus API bearer token used for authenticated API or worker requests."
	case name == "BUS_EVENTS_API_URL":
		return "Base URL for the Bus Events API used by event-backed integrations and providers."
	case name == "BUS_AUTH_API_URL":
		return "Base URL for the Bus Auth API used to issue or validate Bus credentials."
	case name == "BUS_AI_API_URL":
		return "Base URL for the Bus AI API used by AI-facing clients and providers."
	case name == "BUS_API_URL":
		return "Base URL for the Bus API used by this module."
	case strings.HasSuffix(name, "_JWT_SECRET"):
		return "HS256 JWT signing or verification secret for this module's protected API surface."
	case strings.HasSuffix(name, "_HS256_SECRET"):
		return "HS256 JWT signing or verification secret for local Bus authentication."
	case strings.Contains(name, "POSTGRES_DSN") || strings.Contains(name, "DATABASE_URL"):
		return "Database connection string used by this module's persistent backend."
	case strings.Contains(name, "TOKEN"):
		return "Authentication token used by this module."
	case strings.Contains(name, "SECRET"):
		return "Secret value used by this module."
	case strings.Contains(name, "API_URL") || strings.HasSuffix(name, "_URL"):
		return "Service URL used by this module."
	default:
		return ""
	}
}

// knownDescription returns purpose text for Bus environment variables that are
// shared across modules or discovered from module-owned metadata adapters.
//
// Used by: standardDescription.
func knownDescription(name string) (string, bool) {
	descriptions := map[string]string{
		"BUS_AGENT":                              "Command or executable used to launch the Bus agent runtime.",
		"BUS_AGENT_MODEL_REASONING_EFFORT":       "Default reasoning effort passed to Bus agent model sessions.",
		"BUS_AGENT_MODEL_REASONING_SUMMARY":      "Controls whether Bus agent model sessions request reasoning summaries.",
		"BUS_AGENT_MODEL_VERBOSITY":              "Default response verbosity used by Bus agent model sessions.",
		"BUS_API_EVENT_BUS_MODE":                 "Selects the event bus mode used by bus-api when wiring event-backed services.",
		"BUS_BALANCES_APPLY":                     "Controls whether balance e2e or helper flows apply generated balance changes.",
		"BUS_BFL":                                "Path or command used to invoke the bus-bfl module from tests or helper flows.",
		"BUS_BOOKS_TODAY":                        "Overrides the current date used by bus-books deterministic tests and commands.",
		"BUS_CLOUD_ENVIRONMENT":                  "Logical cloud environment name, such as dev or prod, used for cloud resource planning.",
		"BUS_CLOUD_INFERENCE_NODE":               "Cloud node identifier used for inference workloads in cloud deployment plans.",
		"BUS_CLOUD_NETWORK_NAME":                 "Cloud network name used when planning or inspecting Bus cloud resources.",
		"BUS_CLOUD_PROVIDER":                     "Cloud provider selector used by Bus cloud and deployment operators.",
		"BUS_CLOUD_PROXY_NODE":                   "Cloud node identifier used for proxy workloads in cloud deployment plans.",
		"BUS_CONFIG_DIR":                         "Directory containing Bus configuration files and local module state.",
		"BUS_DATA":                               "Path or command used to invoke the bus-data module from tests or helper flows.",
		"BUS_DATABASE_NAMES":                     "Comma- or whitespace-separated database names managed by deployment database steps.",
		"BUS_DATABASE_PROVIDER":                  "Database provider selector used by deployment database operations.",
		"BUS_DATABASE_SCHEMAS":                   "Comma- or whitespace-separated database schema names managed by deployment steps.",
		"BUS_DATABASE_SERVICE_ROLE":              "Database role name granted service access during deployment database setup.",
		"BUS_DATA_TEST_DSN":                      "PostgreSQL DSN used by bus-data tests and local data integration checks.",
		"BUS_DEPLOYMENT_ID":                      "Stable deployment identifier used to name and correlate Bus infrastructure resources.",
		"BUS_DEV_AGENT":                          "Command or executable used as the Bus development agent backend.",
		"BUS_DEV_MODEL_REASONING_EFFORT":         "Reasoning effort passed to model sessions launched by bus-dev.",
		"BUS_DEV_MODEL_REASONING_SUMMARY":        "Controls reasoning-summary requests for model sessions launched by bus-dev.",
		"BUS_DEV_MODEL_VERBOSITY":                "Default model response verbosity used by bus-dev agent commands.",
		"BUS_ENVIRONMENT":                        "Fallback logical environment name used when a module-specific environment variable is unset.",
		"BUS_EXECUTOR":                           "Command runner backend used by bus-reports for executing report workflows.",
		"BUS_FILING_TARGET":                      "Filing target identifier passed from bus-filing to delegated filing commands.",
		"BUS_GATEWAY_LISTEN":                     "Network listen address used by the Bus gateway service.",
		"BUS_GATEWAY_PORT":                       "TCP port used by the Bus gateway service.",
		"BUS_GATEWAY_POSTGRES_SCHEMA":            "PostgreSQL schema used by the Bus gateway storage backend.",
		"BUS_GATEWAY_STORAGE_BACKEND":            "Storage backend selector used by the Bus gateway service.",
		"BUS_GATEWAY_TRUST_SERVICE_ID":           "Trusted service identifier accepted by the Bus gateway for internal calls.",
		"BUS_GATEWAY_WORKSPACE":                  "Workspace path used by the Bus gateway for local files and state.",
		"BUS_INFERENCE_MODEL":                    "Inference model name requested by Bus inference deployment operations.",
		"BUS_INFERENCE_NODE":                     "Node identifier used for Bus inference deployment operations.",
		"BUS_INFERENCE_PROVIDER":                 "Provider selector used by Bus inference operator and deployment commands.",
		"BUS_INIT":                               "Path or command used to invoke the bus-init module from tests or helper flows.",
		"BUS_INVOICES_BIN":                       "Path to the bus-invoices executable used by validation workflows.",
		"BUS_JOURNAL":                            "Path or command used to invoke the bus-journal module from tests or helper flows.",
		"BUS_JOURNAL_BIN":                        "Path to the bus-journal executable used by validation workflows.",
		"BUS_LEDGER_E2E_LOCALE_MATRIX":           "Locale matrix used by bus-ledger e2e tests for user-facing output checks.",
		"BUS_LINT_AGENT":                         "Agent backend executable used by bus-lint for AI-assisted lint checks.",
		"BUS_LINT_OK":                            "Expected success marker used by bus-lint test doubles.",
		"BUS_NODE_ID":                            "Node identifier used by operator deployment commands when targeting one node.",
		"BUS_OPEN_ARGS_FILE":                     "Path to a file containing arguments consumed by ledger open-command tests.",
		"BUS_OPERATOR_CLOUD_ENV_ALLOW":           "Additional environment variable allow-list for bus-operator-cloud commands.",
		"BUS_OPERATOR_DEPLOY_ENV_ALLOW":          "Additional environment variable allow-list for bus-operator-deploy commands.",
		"BUS_OPERATOR_ENV_ALLOW":                 "Shared additional environment variable allow-list for Bus operator commands.",
		"BUS_OPERATOR_INTERNAL_KEY":              "Internal operator shared key used for privileged operator token flows.",
		"BUS_OUTPUT":                             "Output path or mode used by bus-update helper flows.",
		"BUS_PERIOD":                             "Path or command used to invoke the bus-period module from tests or helper flows.",
		"BUS_PERIOD_BIN":                         "Path to the bus-period executable used by validation workflows.",
		"BUS_PORTAL_API_CONNECT_SRC":             "Content Security Policy connect-src value used by the Bus portal frontend.",
		"BUS_PORTAL_ATTACHMENTS_BIN":             "Path to the bus-attachments executable used by portal workflows.",
		"BUS_PORTAL_DEMO_AS_OF":                  "Demo as-of date used by the Bus portal when rendering deterministic demo data.",
		"BUS_PORTAL_DEMO_PERIOD":                 "Demo accounting period used by the Bus portal demo workspace.",
		"BUS_PORTAL_DOCKER_NO_CACHE":             "Controls whether Bus portal Docker builds bypass the Docker build cache.",
		"BUS_PORTAL_FRONTEND_AUTH_REQUIRED":      "Controls whether the Bus portal frontend requires authentication.",
		"BUS_PORTAL_FRONTEND_JWT_AUDIENCE":       "JWT audience expected by the Bus portal frontend.",
		"BUS_PORTAL_FRONTEND_JWT_COOKIE":         "Cookie name used by the Bus portal frontend for JWT authentication.",
		"BUS_PORTAL_FRONTEND_JWT_SCOPE":          "JWT scope required by the Bus portal frontend.",
		"BUS_PORTAL_LISTEN":                      "Network listen address used by the Bus portal service.",
		"BUS_PORTAL_PORT":                        "TCP port used by the Bus portal service.",
		"BUS_PORTAL_REPORTS_BIN":                 "Path to the bus-reports executable used by portal report workflows.",
		"BUS_PORTAL_WORKSPACE":                   "Workspace path served by the Bus portal.",
		"BUS_POSTGRES_ADMIN_DSN_FILE":            "Path to a file containing the PostgreSQL admin DSN for database operator commands.",
		"BUS_PREFERENCES_PATH":                   "Path to the Bus preferences JSON file used for local module preferences.",
		"BUS_REPORTS_EXECUTOR":                   "Executor backend selected for bus-reports report generation.",
		"BUS_RUNNER_APPLY_NETPLAN":               "Controls whether SSH runner setup applies rendered netplan configuration.",
		"BUS_RUNNER_DELETE_POLL_INTERVAL":        "Polling interval used while waiting for runner resources to delete.",
		"BUS_RUNNER_DELETE_READY_STATES":         "Resource states treated as ready for deletion by the runner.",
		"BUS_RUNNER_DELETE_STOP_FIRST":           "Controls whether runner deletion stops resources before deleting them.",
		"BUS_RUNNER_DELETE_STORAGE":              "Controls whether runner deletion also removes attached storage.",
		"BUS_RUNNER_DELETE_TIMEOUT":              "Maximum time to wait for runner resource deletion.",
		"BUS_RUNNER_DNS_SERVERS":                 "DNS server list written into SSH runner network configuration.",
		"BUS_RUNNER_NETPLAN_MODE":                "Netplan rendering mode used by SSH runner setup.",
		"BUS_RUNNER_SSH_READY_POLL_INTERVAL":     "Polling interval used while waiting for SSH readiness.",
		"BUS_RUNNER_SSH_READY_TIMEOUT":           "Maximum time to wait for SSH readiness.",
		"BUS_RUNNER_TRANSIENT_STATES":            "Resource states treated as transient while waiting for runner readiness.",
		"BUS_RUN_AGENT":                          "Command or executable used as the Bus run agent backend.",
		"BUS_RUN_MODEL_REASONING_EFFORT":         "Reasoning effort passed to model sessions launched by bus-run.",
		"BUS_RUN_MODEL_REASONING_SUMMARY":        "Controls reasoning-summary requests for model sessions launched by bus-run.",
		"BUS_RUN_MODEL_VERBOSITY":                "Default model response verbosity used by bus-run agent commands.",
		"BUS_SSH_RUNNER_KNOWN_HOSTS":             "Path to the known_hosts file used by the SSH runner.",
		"BUS_SSH_RUNNER_MAX_OUTPUT_BYTES":        "Maximum command output bytes captured by the SSH runner.",
		"BUS_SSH_RUNNER_PRIVATE_KEY":             "Private SSH key material or path used by the SSH runner.",
		"BUS_SSH_RUNNER_REAL_E2E":                "Enables real SSH runner e2e tests that require external infrastructure.",
		"BUS_STRIPE_LIVE_E2E":                    "Enables live Stripe e2e tests that require real Stripe credentials.",
		"BUS_STUB_FAIL_CODE":                     "Exit code returned by Bus test stub commands when forced to fail.",
		"BUS_STUB_FAIL_MODULE":                   "Module name that Bus test stub commands should force to fail.",
		"BUS_STUB_LOG":                           "Path to the log file written by Bus test stub commands.",
		"BUS_STUB_SKIP_PATH":                     "Path marker used by Bus test stubs to skip selected behavior.",
		"BUS_STUB_WORKDIR":                       "Working directory reported or used by Bus test stub commands.",
		"BUS_SUBCOMMAND":                         "Subcommand name passed by wrapper modules to delegated Bus commands.",
		"BUS_TEST_ACP_CLIENT":                    "Path or command for the ACP client test double used by Bus agent tests.",
		"BUS_TEST_PENDING_APPROVAL_JSON":         "JSON fixture path used to test pending approval behavior.",
		"BUS_TEST_THREAD_ISOLATION_JSON":         "JSON fixture path used to test thread isolation behavior.",
		"BUS_UPCLOUD_PROVIDER":                   "UpCloud provider selector used by UpCloud integration commands.",
		"BUS_UPCLOUD_REAL_E2E":                   "Enables live UpCloud e2e tests that require real UpCloud credentials.",
		"BUS_WORKDIR":                            "Workspace directory passed to delegated Bus commands.",
		"OPENAI_API_KEY":                         "OpenAI API key used by AI-backed Bus integrations and secret management commands.",
		"UPCLOUD_CONTAINER_CODEX_IMAGE":          "Container image used for Codex workers on UpCloud.",
		"UPCLOUD_CONTAINER_NETWORK_MODE":         "Network mode used when running UpCloud containers.",
		"UPCLOUD_CONTAINER_OS":                   "Operating system image used for UpCloud container hosts.",
		"UPCLOUD_CONTAINER_OS_STORAGE_SIZE":      "Storage size allocated for UpCloud container host operating-system disks.",
		"UPCLOUD_CONTAINER_PLAN":                 "UpCloud plan selected for container host resources.",
		"UPCLOUD_CONTAINER_PRIVATE_IP":           "Private IP address assigned to an UpCloud container runner.",
		"UPCLOUD_CONTAINER_PRIVATE_NETWORK_NAME": "Private network name used for UpCloud container runners.",
		"UPCLOUD_CONTAINER_RUNNER_NAME":          "Logical runner name used for UpCloud container execution.",
		"UPCLOUD_CONTAINER_RUN_NETWORK":          "Network attached to UpCloud container run requests.",
		"UPCLOUD_CONTAINER_RUN_READ_ONLY":        "Controls whether UpCloud container run requests use a read-only root filesystem.",
		"UPCLOUD_CONTAINER_RUN_TIMEOUT":          "Maximum duration allowed for UpCloud container run requests.",
		"UPCLOUD_CONTAINER_RUN_TMPFS_SIZE":       "tmpfs size mounted for UpCloud container run requests.",
		"UPCLOUD_CONTAINER_RUN_USER":             "User identity used inside UpCloud container run requests.",
		"UPCLOUD_CONTAINER_RUN_WORKDIR":          "Working directory used inside UpCloud container run requests.",
		"UPCLOUD_CONTAINER_SSH_KEYS":             "SSH public keys installed on UpCloud container hosts.",
		"UPCLOUD_CONTAINER_SSH_TARGET":           "SSH target used to reach an UpCloud container host.",
		"UPCLOUD_CONTAINER_START_TIMEOUT":        "Maximum time to wait for an UpCloud container host to start.",
		"UPCLOUD_CONTAINER_USERNAME_CANDIDATES":  "Candidate SSH usernames tried for UpCloud container hosts.",
		"UPCLOUD_CONTAINER_ZONE":                 "UpCloud zone used for container host resources.",
		"UPCLOUD_VM_NAME":                        "Name assigned to UpCloud virtual machine resources.",
	}
	description, ok := descriptions[name]
	return description, ok
}

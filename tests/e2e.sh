#!/usr/bin/env bash
set -e
[[ "${BUS_E2E_VERBOSE:-0}" = "1" ]] && set -x

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BINARY="${ROOT_DIR}/bin/bus-help"
WORKSPACE="${ROOT_DIR}/tests/e2e_workspace"

cleanup() {
	if [ -z "${BUS_E2E_KEEP:-}" ]; then
		rm -rf "$WORKSPACE"
	fi
}
trap cleanup EXIT

rm -rf "$WORKSPACE"
mkdir -p "$WORKSPACE/bin"

cat > "$WORKSPACE/bin/bus-journal" << 'EOF'
#!/usr/bin/env sh
if [ "$1" = "help" ] && [ "$2" = "--format" ] && [ "$3" = "opencli" ]; then
  printf '%s\n' '{"opencli":"0.1.0","info":{"title":"bus-journal","summary":"Journal help"},"metadata":{"io.busdk.environment":{"version":"0.1","variables":[{"name":"BUS_JOURNAL_ACTOR","description":"Actor","safeHandling":{"printable":true,"storeInDotenv":true,"redactInLogs":false}}]}}}'
  exit 0
fi
printf '%s\n' 'journal help'
EOF
chmod +x "$WORKSPACE/bin/bus-journal"

cd "$WORKSPACE"
PATH="$WORKSPACE/bin:$PATH" "$BINARY" --format opencli journal | grep -q '"opencli": "0.1.0"'
PATH="$WORKSPACE/bin:$PATH" "$BINARY" help --format opencli | grep -q '"title": "bus-help"'
PATH="$WORKSPACE/bin:$PATH" "$BINARY" help --format opencli | grep -q '"io.busdk.environment"'
PATH="$WORKSPACE/bin:$PATH" "$BINARY" env journal | grep -q 'BUS_JOURNAL_ACTOR'
PATH="$WORKSPACE/bin:$PATH" "$BINARY" journal | grep -q 'Environment variables: 1'
echo "PASS bus-help e2e"

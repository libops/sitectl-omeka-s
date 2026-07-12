package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/libops/sitectl/pkg/plugin"
)

func TestCreateDefinitionLifecycleContract(t *testing.T) {
	t.Parallel()
	spec := createDefinition()
	if len(spec.Images) != 1 || spec.Images[0].Image != "libops/omeka-s:4.2.1-php84" || spec.Images[0].BuildPolicy != plugin.BuildPolicyAlways {
		t.Fatalf("unexpected Omeka S image contract: %+v", spec.Images)
	}
	if len(spec.DockerComposeUp) != 1 || !strings.Contains(spec.DockerComposeUp[0], "--wait --wait-timeout 600") {
		t.Fatalf("create must wait for service health before reporting ready: %+v", spec.DockerComposeUp)
	}
	assertManualMigrationRollout(t, spec.DockerComposeRollout, "omeka-s", "*/migrate")

	sdk := plugin.NewSDK(plugin.Metadata{Name: "omeka-s"})
	RegisterCommands(sdk)
	for _, definition := range sdk.LocalComponentDefinitions() {
		if definition.Name == "dev-mode" {
			t.Fatal("dev-mode must not mask bundled Omeka S extension directories")
		}
	}
}

func assertManualMigrationRollout(t *testing.T, commands []string, service, migrationMarker string) {
	t.Helper()

	if len(commands) != 8 {
		t.Fatalf("rollout commands = %+v, want eight explicit steps", commands)
	}
	if !strings.HasPrefix(commands[0], "docker compose pull ") || !strings.HasPrefix(commands[1], "docker compose build ") {
		t.Fatalf("rollout must prepare pulls and builds before the outage: %+v", commands)
	}
	initialStart := commands[4]
	if initialStart != "docker compose up --remove-orphans --pull missing --quiet-pull -d "+service || strings.Contains(initialStart, "--wait") {
		t.Fatalf("migration inspection must start only %s: %q", service, initialStart)
	}
	if !strings.Contains(commands[5], "until test -f /installed") || !strings.Contains(commands[5], "deadline=") || !strings.Contains(commands[5], "+ 600") || !strings.Contains(commands[5], "--max-time 5") || !strings.Contains(commands[5], "/status") {
		t.Fatalf("migration inspection readiness must be bounded: %q", commands[5])
	}
	gate := commands[6]
	if !strings.HasPrefix(gate, "docker compose exec -T "+service+" ") {
		t.Fatalf("migration gate must begin with a rewritable Compose command: %q", gate)
	}
	for _, required := range []string{migrationMarker, "--connect-timeout 2", "--max-time 30", "ACTION REQUIRED", "Public Traefik remains stopped", "sitectl port-forward", "sitectl deploy --skip-git --no-pull", "same --context NAME", "exit 10"} {
		if !strings.Contains(gate, required) {
			t.Fatalf("manual migration gate missing %q: %q", required, gate)
		}
	}
	if strings.Contains(gate, "|| true") {
		t.Fatalf("manual migration inspection must fail hard: %q", gate)
	}
	finalStart := commands[7]
	if finalStart != "docker compose up --remove-orphans --wait --wait-timeout 600 --pull missing --quiet-pull -d" {
		t.Fatalf("bounded full-stack start must run only after migration is current: %q", finalStart)
	}
	for _, command := range commands {
		if output, err := exec.Command("bash", "-n", "-c", command).CombinedOutput(); err != nil {
			t.Fatalf("rollout command has invalid shell syntax: %v\n%s\n%s", err, output, command)
		}
		assertNestedShellSyntax(t, command)
	}
}

func assertNestedShellSyntax(t *testing.T, command string) {
	t.Helper()
	const marker = "sh -c '"
	start := strings.Index(command, marker)
	if start == -1 {
		return
	}
	if !strings.HasSuffix(command, "'") {
		t.Fatalf("nested shell command is not single-quote terminated: %q", command)
	}
	script := command[start+len(marker) : len(command)-1]
	if output, err := exec.Command("sh", "-n", "-c", script).CombinedOutput(); err != nil {
		t.Fatalf("nested shell command has invalid syntax: %v\n%s\n%s", err, output, script)
	}
}

func TestManualMigrationGateBehavior(t *testing.T) {
	t.Parallel()
	gate := createDefinition().DockerComposeRollout[6]
	tests := []struct {
		name       string
		curlBody   string
		curlExit   int
		wantExit   int
		wantOutput string
	}{
		{name: "database current", curlBody: "302 http://127.0.0.1/admin/login", wantExit: 0},
		{name: "migration required", curlBody: "302 http://127.0.0.1/migrate", wantExit: 10, wantOutput: "ACTION REQUIRED"},
		{name: "unexpected response", curlBody: "500 ", wantExit: 3, wantOutput: "Unexpected Omeka S admin response"},
		{name: "curl failure", curlExit: 28, wantExit: 28, wantOutput: "curl status 28"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			env, _ := installRolloutStubs(t, test.curlBody, test.curlExit)
			command := exec.Command("bash", "-c", gate)
			command.Env = env
			output, err := command.CombinedOutput()
			if got := commandExitCode(t, err); got != test.wantExit {
				t.Fatalf("gate exit = %d, want %d; output:\n%s", got, test.wantExit, output)
			}
			if test.wantOutput != "" && !strings.Contains(string(output), test.wantOutput) {
				t.Fatalf("gate output missing %q:\n%s", test.wantOutput, output)
			}
		})
	}
}

func TestMigrationReadinessUsesWallClockDeadline(t *testing.T) {
	t.Parallel()
	readiness := createDefinition().DockerComposeRollout[5]
	env, logPath := installRolloutStubs(t, "", 0)
	dateState := filepath.Join(filepath.Dir(logPath), "date.state")
	dateStub := `#!/bin/sh
set -eu
if [ ! -e "$FAKE_DATE_STATE" ]; then
  : >"$FAKE_DATE_STATE"
  printf '%s\n' 100
else
  printf '%s\n' 701
fi
`
	if err := os.WriteFile(filepath.Join(filepath.Dir(logPath), "date"), []byte(dateStub), 0o755); err != nil {
		t.Fatal(err)
	}
	env = append(env, "FAKE_EXECUTE_READINESS=1", "FAKE_DATE_STATE="+dateState)
	command := exec.Command("bash", "-c", readiness)
	command.Env = env
	output, err := command.CombinedOutput()
	if got := commandExitCode(t, err); got != 1 {
		t.Fatalf("readiness exit = %d, want 1; output:\n%s", got, output)
	}
	if !strings.Contains(string(output), "within 10 minutes") {
		t.Fatalf("readiness did not report its wall-clock deadline:\n%s", output)
	}
}

func TestManualMigrationStopsBeforeFullStackStart(t *testing.T) {
	t.Parallel()
	commands := createDefinition().DockerComposeRollout
	env, logPath := installRolloutStubs(t, "302 http://127.0.0.1/migrate", 0)

	var gateOutput []byte
	gateExit := 0
	for _, commandText := range commands[4:] {
		command := exec.Command("bash", "-c", commandText)
		command.Env = env
		output, err := command.CombinedOutput()
		if err != nil {
			gateOutput = output
			gateExit = commandExitCode(t, err)
			break
		}
	}
	if gateExit != 10 || !strings.Contains(string(gateOutput), "ACTION REQUIRED") {
		t.Fatalf("migration gate did not stop rollout: exit=%d output=%s", gateExit, gateOutput)
	}
	calls, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(calls), "up --remove-orphans --pull missing --quiet-pull -d omeka-s") {
		t.Fatalf("app-only start was not attempted:\n%s", calls)
	}
	if strings.Contains(string(calls), "up --remove-orphans --wait --wait-timeout 600") {
		t.Fatalf("full-stack start ran after a failed migration gate:\n%s", calls)
	}
}

func commandExitCode(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("command failed without an exit status: %v", err)
	}
	return exitErr.ExitCode()
}

func installRolloutStubs(t *testing.T, curlBody string, curlExit int) ([]string, string) {
	t.Helper()
	binDir := t.TempDir()
	logPath := filepath.Join(binDir, "docker.log")
	dockerStub := `#!/bin/sh
set -eu
printf '%s\n' "$*" >>"$FAKE_DOCKER_LOG"
if [ "${1:-}" != compose ]; then
  exit 64
fi
case "${2:-}" in
  up|pull|build|run)
    exit 0
    ;;
  exec)
    case "$*" in
      *http://127.0.0.1/status*)
        if [ "${FAKE_EXECUTE_READINESS:-0}" -ne 1 ]; then
          exit 0
        fi
        ;;
    esac
    shift 4
    exec "$@"
    ;;
  *)
    exit 64
    ;;
esac
`
	curlStub := `#!/bin/sh
set -eu
if [ "${FAKE_CURL_EXIT:-0}" -ne 0 ]; then
  exit "$FAKE_CURL_EXIT"
fi
printf '%s' "${FAKE_CURL_BODY:-}"
`
	for name, content := range map[string]string{"docker": dockerStub, "curl": curlStub} {
		if err := os.WriteFile(filepath.Join(binDir, name), []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	env := append([]string{}, os.Environ()...)
	env = append(env,
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE_DOCKER_LOG="+logPath,
		"FAKE_CURL_BODY="+curlBody,
		fmt.Sprintf("FAKE_CURL_EXIT=%d", curlExit),
	)
	return env, logPath
}

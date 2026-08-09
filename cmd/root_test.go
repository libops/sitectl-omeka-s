package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/libops/sitectl/pkg/plugin"
)

func TestCreateDefinitionLifecycleContract(t *testing.T) {
	t.Parallel()
	spec := createDefinition()
	if spec.DockerComposeRepo != createRepo || spec.DockerComposeBranch != omekaSTemplateVersion {
		t.Fatalf("unexpected immutable template source: %s@%s", spec.DockerComposeRepo, spec.DockerComposeBranch)
	}
	if spec.DockerComposeBranch != "v1.2.1" {
		t.Fatalf("Omeka S template revision = %q, want immutable v1.2.1", spec.DockerComposeBranch)
	}
	if len(spec.Images) != 1 || spec.Images[0].Image != "libops/omeka-s:4.2.1-php84" || spec.Images[0].BuildPolicy != plugin.BuildPolicyAlways {
		t.Fatalf("unexpected Omeka S image contract: %+v", spec.Images)
	}
	if len(spec.DockerComposeInit) != 3 || spec.DockerComposeInit[2] != omekaSRolloutPreflightCommand {
		t.Fatalf("create must validate the checked-in runtime programs after initialization: %+v", spec.DockerComposeInit)
	}
	if len(spec.DockerComposeUp) != 1 || !strings.Contains(spec.DockerComposeUp[0], "--wait --wait-timeout 600") {
		t.Fatalf("create must wait for service health before reporting ready: %+v", spec.DockerComposeUp)
	}
	assertManualMigrationRollout(t, spec.DockerComposeRollout)

	sdk := plugin.NewSDK(plugin.Metadata{Name: pluginName})
	RegisterCommands(sdk)
	for _, definition := range sdk.LocalComponentDefinitions() {
		if definition.Name == "dev-mode" {
			t.Fatal("dev-mode must not mask bundled Omeka S extension directories")
		}
	}
	if len(sdk.DeployDefinitions()) != 1 || sdk.DeployDefinitions()[0].Name != "default" {
		t.Fatalf("expected the Omeka S template compatibility deploy hook, got %+v", sdk.DeployDefinitions())
	}
}

func assertManualMigrationRollout(t *testing.T, commands []string) {
	t.Helper()

	if len(commands) != 8 {
		t.Fatalf("rollout commands = %+v, want eight explicit steps", commands)
	}
	if !strings.HasPrefix(commands[0], "docker compose pull ") || !strings.HasPrefix(commands[1], "docker compose build ") {
		t.Fatalf("rollout must prepare pulls and builds before the outage: %+v", commands)
	}
	if commands[4] != "docker compose up --remove-orphans --pull missing --quiet-pull -d "+omekaSService || strings.Contains(commands[4], "--wait") {
		t.Fatalf("migration inspection must start only %s: %q", omekaSService, commands[4])
	}
	if commands[5] != "docker compose exec -T omeka-s "+omekaSRolloutReadinessTarget {
		t.Fatalf("rollout must invoke the checked-in bounded readiness program: %q", commands[5])
	}
	if commands[6] != "docker compose exec -T omeka-s "+omekaSMigrationGateTarget {
		t.Fatalf("rollout must invoke the checked-in migration gate: %q", commands[6])
	}
	if commands[7] != "docker compose up --remove-orphans --wait --wait-timeout 600 --pull missing --quiet-pull -d" {
		t.Fatalf("bounded full-stack start must run only after migration is current: %q", commands[7])
	}
}

func TestProductionSourcesDoNotEmbedRuntimePrograms(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"root.go", "verify.go"} {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", name, err)
		}
		for _, forbidden := range []string{
			`[]string{"php", "-r"`,
			"php -r",
			`[]string{"bash", "-lc"`,
			"bash -lc",
			`[]string{"sh", "-c"`,
			"sh -c",
			"SELECT CURRENT_USER",
			"until test -f /installed",
		} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("%s embeds forbidden runtime program fragment %q", name, forbidden)
			}
		}
	}
}

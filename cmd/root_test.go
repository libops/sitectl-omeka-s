package cmd

import (
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
	rollout := strings.Join(spec.DockerComposeRollout, "\n")
	if !strings.Contains(rollout, "docker compose build --pull") || !strings.Contains(rollout, "ACTION REQUIRED") || !strings.Contains(rollout, "/admin") || strings.Contains(rollout, "|| true") || strings.Contains(rollout, "--wait") {
		t.Fatalf("rollout must rebuild, propagate failures, and require the supported admin migration:\n%s", rollout)
	}

	sdk := plugin.NewSDK(plugin.Metadata{Name: "omeka-s"})
	RegisterCommands(sdk)
	for _, definition := range sdk.LocalComponentDefinitions() {
		if definition.Name == "dev-mode" {
			t.Fatal("dev-mode must not mask bundled Omeka S extension directories")
		}
	}
}

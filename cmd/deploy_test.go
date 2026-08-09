package cmd

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/libops/sitectl/pkg/config"
	"github.com/libops/sitectl/pkg/plugin"
	"github.com/spf13/cobra"
)

func TestOmekaSDeployRunnerPreDownChecksTrackedAndEffectivePrograms(t *testing.T) {
	projectDir := writeOmekaSProgramContractFixture(t, omekaSProgramComposeFixture(true))
	ctx := &config.Context{DockerHostType: config.ContextLocal, ProjectDir: projectDir}
	sdk := plugin.NewSDK(plugin.Metadata{Name: pluginName})

	oldRun := runOmekaSComposeProjectCommand
	t.Cleanup(func() { runOmekaSComposeProjectCommand = oldRun })
	var commands []string
	runOmekaSComposeProjectCommand = func(gotSDK *plugin.SDK, _ context.Context, gotCtx *config.Context, gotProjectDir string, _, _ io.Writer, command string) error {
		if gotSDK != sdk || gotCtx != ctx || gotProjectDir != projectDir {
			t.Fatalf("unexpected deploy context: sdk=%p ctx=%#v project=%q", gotSDK, gotCtx, gotProjectDir)
		}
		commands = append(commands, command)
		return nil
	}

	cmd := &cobra.Command{Use: "pre-down"}
	cmd.SetContext(t.Context())
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	if err := (&omekaSDeployRunner{sdk: sdk}).PreDown(cmd, ctx); err != nil {
		t.Fatalf("PreDown() error = %v", err)
	}

	want := []string{omekaSRolloutPreflightCommand, omekaSComposeConfigCommand}
	for _, program := range omekaSTemplatePrograms {
		probe := "-r"
		if program.executable {
			probe = "-x"
		}
		want = append(want, "docker compose run --rm --no-deps --entrypoint test omeka-s "+probe+" "+program.target)
	}
	if strings.Join(commands, "\n") != strings.Join(want, "\n") {
		t.Fatalf("effective Compose checks = %#v, want %#v", commands, want)
	}
}

func TestOmekaSDeployRunnerPreDownFailsClearlyWhenTrackedPreflightRejects(t *testing.T) {
	projectDir := writeOmekaSProgramContractFixture(t, omekaSProgramComposeFixture(true))
	ctx := &config.Context{DockerHostType: config.ContextLocal, ProjectDir: projectDir}

	oldRun := runOmekaSComposeProjectCommand
	t.Cleanup(func() { runOmekaSComposeProjectCommand = oldRun })
	runOmekaSComposeProjectCommand = func(_ *plugin.SDK, _ context.Context, _ *config.Context, _ string, _, _ io.Writer, command string) error {
		if command == omekaSRolloutPreflightCommand {
			return errors.New("rejected source")
		}
		return nil
	}

	err := (&omekaSDeployRunner{sdk: plugin.NewSDK(plugin.Metadata{Name: pluginName})}).PreDown(&cobra.Command{Use: "pre-down"}, ctx)
	assertOmekaSTemplateMigrationError(t, err, "checked-in preflight")
}

func TestValidateOmekaSTemplateProgramsAcceptsReadOnlyMounts(t *testing.T) {
	t.Parallel()
	projectDir := writeOmekaSProgramContractFixture(t, omekaSProgramComposeFixture(true))
	ctx := &config.Context{DockerHostType: config.ContextLocal, ProjectDir: projectDir}
	if err := validateOmekaSTemplatePrograms(t.Context(), ctx); err != nil {
		t.Fatalf("validateOmekaSTemplatePrograms() error = %v", err)
	}
}

func TestValidateOmekaSTemplateProgramsRejectsOlderCheckout(t *testing.T) {
	t.Parallel()
	projectDir := writeOmekaSProgramContractFixture(t, omekaSProgramComposeFixture(true))
	if err := os.Remove(filepath.Join(projectDir, filepath.FromSlash(omekaSVersionSource))); err != nil {
		t.Fatalf("Remove(program) error = %v", err)
	}
	ctx := &config.Context{DockerHostType: config.ContextLocal, ProjectDir: projectDir}
	assertOmekaSTemplateMigrationError(t, validateOmekaSTemplatePrograms(t.Context(), ctx), "missing checked-in "+omekaSVersionSource)
}

func TestValidateOmekaSTemplateProgramsRejectsWritableMount(t *testing.T) {
	t.Parallel()
	projectDir := writeOmekaSProgramContractFixture(t, omekaSProgramComposeFixture(false))
	ctx := &config.Context{DockerHostType: config.ContextLocal, ProjectDir: projectDir}
	assertOmekaSTemplateMigrationError(t, validateOmekaSTemplatePrograms(t.Context(), ctx), "without the required read-only checked-in bind")
}

func TestValidateOmekaSTemplateProgramsRejectsSymlink(t *testing.T) {
	t.Parallel()
	projectDir := writeOmekaSProgramContractFixture(t, omekaSProgramComposeFixture(true))
	programPath := filepath.Join(projectDir, filepath.FromSlash(omekaSRolloutReadinessSource))
	if err := os.Remove(programPath); err != nil {
		t.Fatalf("Remove(program) error = %v", err)
	}
	if err := os.Symlink(filepath.Join(projectDir, "compose.yaml"), programPath); err != nil {
		t.Fatalf("Symlink(program) error = %v", err)
	}
	ctx := &config.Context{DockerHostType: config.ContextLocal, ProjectDir: projectDir}
	assertOmekaSTemplateMigrationError(t, validateOmekaSTemplatePrograms(t.Context(), ctx), "must be a regular checked-in file")
}

func TestValidateOmekaSTemplateProgramsIgnoresUnconfiguredLegacyComposeFile(t *testing.T) {
	t.Parallel()
	projectDir := writeOmekaSProgramContractFixture(t, omekaSProgramComposeFixture(true))
	if err := os.WriteFile(filepath.Join(projectDir, "docker-compose.yaml"), []byte(omekaSProgramComposeFixture(false)), 0o644); err != nil {
		t.Fatalf("WriteFile(docker-compose.yaml) error = %v", err)
	}
	ctx := &config.Context{
		DockerHostType: config.ContextLocal,
		ProjectDir:     projectDir,
		ComposeFile:    []string{"compose.yaml"},
	}
	if err := validateOmekaSTemplatePrograms(t.Context(), ctx); err != nil {
		t.Fatalf("validateOmekaSTemplatePrograms() inspected an unconfigured Compose file: %v", err)
	}
}

func writeOmekaSProgramContractFixture(t *testing.T, compose string) string {
	t.Helper()
	projectDir := t.TempDir()
	for _, program := range append([]omekaSTemplateProgram{{source: omekaSRolloutPreflightSource, executable: true}}, omekaSTemplatePrograms...) {
		path := filepath.Join(projectDir, filepath.FromSlash(program.source))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", path, err)
		}
		mode := os.FileMode(0o644)
		if program.executable {
			mode = 0o755
		}
		if err := os.WriteFile(path, []byte("fixture\n"), mode); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", path, err)
		}
	}
	if err := os.WriteFile(filepath.Join(projectDir, "compose.yaml"), []byte(compose), 0o644); err != nil {
		t.Fatalf("WriteFile(compose.yaml) error = %v", err)
	}
	return projectDir
}

func omekaSProgramComposeFixture(readOnly bool) string {
	mode := "ro,z"
	if !readOnly {
		mode = "rw,z"
	}
	var lines []string
	for _, program := range omekaSTemplatePrograms {
		lines = append(lines, "      - ./"+program.source+":"+program.target+":"+mode)
	}
	return "services:\n  omeka-s:\n    volumes:\n" + strings.Join(lines, "\n") + "\n"
}

func assertOmekaSTemplateMigrationError(t *testing.T, err error, detail string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected Omeka S template compatibility failure")
	}
	for _, want := range []string{
		"before services were stopped",
		detail,
		createRepo,
		omekaSTemplateVersion,
		"rerun sitectl deploy",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %v", want, err)
		}
	}
}

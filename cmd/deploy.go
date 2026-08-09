package cmd

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/libops/sitectl/pkg/config"
	"github.com/libops/sitectl/pkg/plugin"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var omekaSDefaultComposeFiles = []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"}

var runOmekaSComposeProjectCommand = func(sdk *plugin.SDK, runCtx context.Context, ctx *config.Context, projectDir string, stdout, stderr io.Writer, command string) error {
	if sdk == nil {
		return fmt.Errorf("plugin SDK for Omeka S is not initialized")
	}
	return sdk.RunComposeProjectCommandContext(runCtx, ctx, projectDir, stdout, stderr, command)
}

type omekaSDeployRunner struct {
	sdk *plugin.SDK
}

func (*omekaSDeployRunner) BindFlags(*cobra.Command) {}

func (r *omekaSDeployRunner) PreDown(cmd *cobra.Command, ctx *config.Context) error {
	runCtx := context.Background()
	if cmd != nil && cmd.Context() != nil {
		runCtx = cmd.Context()
	}
	if err := validateOmekaSTemplatePrograms(runCtx, ctx); err != nil {
		return err
	}

	stdout, stderr := io.Writer(io.Discard), io.Writer(io.Discard)
	if cmd != nil {
		stdout = cmd.OutOrStdout()
		stderr = cmd.ErrOrStderr()
	}
	if err := runOmekaSComposeProjectCommand(r.sdk, runCtx, ctx, ctx.ProjectDir, stdout, stderr, omekaSRolloutPreflightCommand); err != nil {
		return omekaSTemplateMigrationError(fmt.Sprintf("the checked-in preflight rejected a required runtime program: %v", err))
	}
	if err := runOmekaSComposeProjectCommand(r.sdk, runCtx, ctx, ctx.ProjectDir, stdout, stderr, omekaSComposeConfigCommand); err != nil {
		return fmt.Errorf("validate effective Omeka S Compose configuration before services were stopped: %w", err)
	}
	for _, program := range omekaSTemplatePrograms {
		probe := "-r"
		if program.executable {
			probe = "-x"
		}
		command := fmt.Sprintf("docker compose run --rm --no-deps --entrypoint test omeka-s %s %s", probe, program.target)
		if err := runOmekaSComposeProjectCommand(r.sdk, runCtx, ctx, ctx.ProjectDir, stdout, stderr, command); err != nil {
			return omekaSTemplateMigrationError(fmt.Sprintf("the effective omeka-s service cannot use required program %s", program.target))
		}
	}
	return nil
}

func (*omekaSDeployRunner) PostUp(*cobra.Command, *config.Context) error {
	return nil
}

func omekaSDeployDefinition() plugin.DeploySpec {
	return plugin.DeploySpec{
		Name:        "default",
		Description: "Validate the versioned Omeka S template programs before replacing running services",
		Default:     true,
	}
}

func validateOmekaSTemplatePrograms(runCtx context.Context, ctx *config.Context) error {
	if ctx == nil {
		return fmt.Errorf("validate Omeka S runtime programs: context is nil")
	}
	projectDir := strings.TrimSpace(ctx.ProjectDir)
	if projectDir == "" {
		return fmt.Errorf("validate Omeka S runtime programs: context %q does not define a project directory", ctx.Name)
	}

	required := append([]omekaSTemplateProgram{{source: omekaSRolloutPreflightSource, executable: true}}, omekaSTemplatePrograms...)
	for _, program := range required {
		programPath := filepath.Join(projectDir, filepath.FromSlash(program.source))
		exists, err := ctx.FileExists(programPath)
		if err != nil {
			return fmt.Errorf("inspect Omeka S template program %q: %w", program.source, err)
		}
		if !exists {
			return omekaSTemplateMigrationError("missing checked-in " + program.source)
		}
		for _, probeArgs := range [][]string{{"-f", programPath}, {"!", "-L", programPath}} {
			if _, err := ctx.RunQuietCommandContext(runCtx, exec.Command("test", probeArgs...)); err != nil { // #nosec G204 -- fixed file probes receive a context-scoped path as a separate argument.
				return omekaSTemplateMigrationError(program.source + " must be a regular checked-in file, not a directory or symbolic link")
			}
		}
		if program.executable {
			if _, err := ctx.RunQuietCommandContext(runCtx, exec.Command("test", "-x", programPath)); err != nil { // #nosec G204 -- fixed file probe receives a context-scoped path as a separate argument.
				return omekaSTemplateMigrationError(program.source + " must be executable")
			}
		}
	}

	foundComposeFile := false
	foundPrograms := make(map[string]bool, len(omekaSTemplatePrograms))
	for _, composePath := range omekaSComposePaths(ctx) {
		exists, err := ctx.FileExists(composePath)
		if err != nil {
			return fmt.Errorf("inspect Compose file %q: %w", composePath, err)
		}
		if !exists {
			continue
		}
		foundComposeFile = true
		data, err := ctx.ReadFile(composePath)
		if err != nil {
			return fmt.Errorf("read Compose file %q: %w", composePath, err)
		}
		for _, program := range omekaSTemplatePrograms {
			matches, conflicts, err := composeHasReadOnlyOmekaSProgram(data, projectDir, program)
			if err != nil {
				return fmt.Errorf("parse Compose file %q: %w", composePath, err)
			}
			if conflicts {
				return omekaSTemplateMigrationError(fmt.Sprintf("Compose file %s overrides %s without the required read-only checked-in bind", composePath, program.target))
			}
			foundPrograms[program.target] = foundPrograms[program.target] || matches
		}
	}
	if !foundComposeFile {
		return omekaSTemplateMigrationError("no configured Compose file was found")
	}
	for _, program := range omekaSTemplatePrograms {
		if !foundPrograms[program.target] {
			return omekaSTemplateMigrationError("the omeka-s service does not bind every required checked-in runtime program read-only at its stable target")
		}
	}
	return nil
}

func omekaSComposePaths(ctx *config.Context) []string {
	candidates := append([]string{}, ctx.ComposeFile...)
	if len(candidates) == 0 {
		candidates = append(candidates, omekaSDefaultComposeFiles...)
	}
	paths := make([]string, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		resolved := candidate
		if !filepath.IsAbs(resolved) {
			resolved = filepath.Join(ctx.ProjectDir, resolved)
		}
		resolved = filepath.Clean(resolved)
		if _, ok := seen[resolved]; ok {
			continue
		}
		seen[resolved] = struct{}{}
		paths = append(paths, resolved)
	}
	return paths
}

func composeHasReadOnlyOmekaSProgram(data []byte, projectDir string, program omekaSTemplateProgram) (bool, bool, error) {
	var compose struct {
		Services map[string]struct {
			Volumes []yaml.Node `yaml:"volumes"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return false, false, err
	}
	service, ok := compose.Services[omekaSService]
	if !ok {
		return false, false, nil
	}
	found := false
	for _, volume := range service.Volumes {
		targetsProgram, matches := composeVolumeMatchesOmekaSProgram(volume, projectDir, program)
		if targetsProgram && !matches {
			return false, true, nil
		}
		found = found || matches
	}
	return found, false, nil
}

func composeVolumeMatchesOmekaSProgram(volume yaml.Node, projectDir string, program omekaSTemplateProgram) (bool, bool) {
	if volume.Kind == yaml.AliasNode && volume.Alias != nil {
		return composeVolumeMatchesOmekaSProgram(*volume.Alias, projectDir, program)
	}
	switch volume.Kind {
	case yaml.ScalarNode:
		parts := strings.SplitN(strings.TrimSpace(volume.Value), ":", 3)
		if len(parts) < 2 || filepath.Clean(parts[1]) != program.target {
			return false, false
		}
		return true, len(parts) == 3 && omekaSProgramSourceMatches(parts[0], projectDir, program.source) && composeVolumeModeIsReadOnly(parts[2])
	case yaml.MappingNode:
		values := make(map[string]string, len(volume.Content)/2)
		for index := 0; index+1 < len(volume.Content); index += 2 {
			values[strings.TrimSpace(volume.Content[index].Value)] = strings.TrimSpace(volume.Content[index+1].Value)
		}
		if filepath.Clean(values["target"]) != program.target {
			return false, false
		}
		volumeType := values["type"]
		return true, (volumeType == "" || volumeType == "bind") && strings.EqualFold(values["read_only"], "true") &&
			omekaSProgramSourceMatches(values["source"], projectDir, program.source) &&
			filepath.Clean(values["target"]) == program.target
	default:
		return false, false
	}
}

func composeVolumeModeIsReadOnly(mode string) bool {
	for _, option := range strings.Split(mode, ",") {
		if strings.TrimSpace(option) == "ro" {
			return true
		}
	}
	return false
}

func omekaSProgramSourceMatches(source, projectDir, expectedSource string) bool {
	source = strings.TrimSpace(source)
	if source == "" {
		return false
	}
	if !filepath.IsAbs(source) {
		source = filepath.Join(projectDir, source)
	}
	want := filepath.Join(projectDir, filepath.FromSlash(expectedSource))
	return filepath.Clean(source) == filepath.Clean(want)
}

func omekaSTemplateMigrationError(detail string) error {
	return fmt.Errorf(
		"template compatibility check for Omeka S failed before services were stopped: %s; update the site checkout from %s at %s or newer so its checked-in read-only runtime programs match the plugin contract, then rerun sitectl deploy",
		detail,
		createRepo,
		omekaSTemplateVersion,
	)
}

var _ plugin.DeployRunner = (*omekaSDeployRunner)(nil)

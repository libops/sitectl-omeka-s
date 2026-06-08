package cmd

import "github.com/libops/sitectl/pkg/plugin"

const (
	createRepo   = "https://github.com/libops/omeka-s"
	createBranch = "main"
	pluginName   = "omeka-s"
	defaultPath  = "./omeka-s"
	displayName  = "Omeka S"
)

func createDefinition() plugin.CreateSpec {
	return plugin.CreateSpec{
		Name:                 "default",
		Description:          "Create an Omeka S stack",
		Default:              true,
		MinCPUCores:          2,
		MinMemory:            "4 GiB",
		MinDiskSpace:         "20 GiB",
		DockerComposeRepo:    createRepo,
		DockerComposeBranch:  createBranch,
		DockerComposeBuild:   []string{"make build"},
		DockerComposeInit:    []string{"make init"},
		DockerComposeUp:      []string{"make up"},
		DockerComposeDown:    []string{"make down"},
		DockerComposeRollout: []string{"make rollout"},
	}
}

// RegisterCommands registers Omeka S commands with the plugin SDK.
func RegisterCommands(s *plugin.SDK) {
	s.SetComposeProjectDiscovery(plugin.ComposeProjectDiscovery{
		RequiredServices: []string{"omeka-s"},
		Reason:           "omeka-s service",
	})
	s.RegisterStandardComposeTemplate(createDefinition(), plugin.StandardComposeTemplateOptions{
		DefaultPath:   defaultPath,
		DefaultPlugin: pluginName,
		ReadyMessage:  "Omeka S is ready for use through sitectl.",
		DisplayName:   displayName,
	})
	registerOmekaSCommands(s)
}

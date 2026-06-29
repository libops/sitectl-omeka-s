package cmd

import (
	corecomponent "github.com/libops/sitectl/pkg/component"
	"github.com/libops/sitectl/pkg/plugin"
	coredevmode "github.com/libops/sitectl/pkg/services/devmode"
	coretraefik "github.com/libops/sitectl/pkg/services/traefik"
)

const (
	createRepo   = "https://github.com/libops/omeka-s"
	createBranch = "main"
	pluginName   = "omeka-s"
	defaultPath  = "./omeka-s"
)

func createDefinition() plugin.CreateSpec {
	return plugin.CreateSpec{
		Name:                "default",
		Description:         "Create an Omeka S stack",
		Default:             true,
		MinCPUCores:         2,
		MinMemory:           "4 GiB",
		MinDiskSpace:        "20 GiB",
		DockerComposeRepo:   createRepo,
		DockerComposeBranch: createBranch,
		DockerComposeBuild: []string{
			"docker compose pull --ignore-buildable",
			"docker compose build",
		},
		Images: []plugin.ComposeImageSpec{
			{Service: "omeka-s", Image: "libops/omeka-s:nginx-1.30.3-php84", BuildPolicy: plugin.BuildPolicyIfNotPresent},
		},
		DockerComposeInit: []string{
			"docker compose run --rm init",
		},
		InitArtifacts: []plugin.InitArtifact{
			{Path: "secrets/DB_ROOT_PASSWORD"},
			{Path: "secrets/OMEKA_S_DB_PASSWORD"},
			{Path: "secrets/OMEKA_S_ADMIN_PASSWORD"},
		},
		InitVolumes: []plugin.InitVolume{
			{Name: "mariadb-data"},
			{Name: "omeka-s-files"},
		},
		DockerComposeUp: []string{
			"docker compose up --remove-orphans -d",
		},
		DockerComposeDown: []string{"docker compose down"},
		DockerComposeRollout: []string{
			"docker compose pull --ignore-buildable --quiet || docker compose pull --ignore-buildable || true",
			"docker compose build --pull",
			"docker compose run --rm init",
			"docker compose up --remove-orphans --wait --pull missing --quiet-pull -d",
		},
	}
}

// RegisterCommands registers Omeka S commands with the plugin SDK.
func RegisterCommands(s *plugin.SDK) {
	s.SetComposeProjectDiscovery(plugin.ComposeProjectDiscovery{
		RequiredServices: []string{"omeka-s"},
		Reason:           "omeka-s service",
	})
	s.RegisterComposeTemplateCreateRunner(createDefinition(), plugin.ComposeTemplateCreateOptions{
		DefaultPath:   defaultPath,
		DefaultPlugin: pluginName,
		ReadyMessage:  "Omeka S is ready for use through sitectl.",
	})
	registerApplicationComponents(s, "Omeka S", "omeka-s")
	s.RegisterHealthcheckRunner(omekaSHealthcheckRunner)
	registerOmekaSCommands(s)
}

func registerApplicationComponents(s *plugin.SDK, displayName, appService string) {
	ingress, err := coretraefik.Ingress(coretraefik.IngressOptions{
		AppService:      appService,
		HTTPEntrypoint:  "web",
		HTTPSEntrypoint: "websecure",
		ServiceEnvTemplates: map[string]map[string]string{
			appService: {
				"DOMAIN":               "{domain}",
				"OMEKA_S_ENABLE_HTTPS": "{https_enabled}",
			},
		},
	})
	if err != nil {
		panic(err)
	}
	devMode, err := coredevmode.Component(coredevmode.Options{
		AppService: appService,
		Volumes: []string{
			"./modules:/var/www/omeka-s/modules:z,rw",
			"./themes:/var/www/omeka-s/themes:z,rw",
		},
	})
	if err != nil {
		panic(err)
	}
	s.RegisterServiceComponents(plugin.ServiceComponentRegistryOptions{
		DisplayName: displayName,
		Components:  []corecomponent.ComposeServiceComponent{ingress, devMode},
	})
}

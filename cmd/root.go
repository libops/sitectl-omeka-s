package cmd

import (
	corecomponent "github.com/libops/sitectl/pkg/component"
	"github.com/libops/sitectl/pkg/plugin"
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
			{Service: "omeka-s", Image: "libops/omeka-s:4.2.1-php84", BuildPolicy: plugin.BuildPolicyAlways},
		},
		DockerComposeInit: []string{
			"mkdir -p ./secrets",
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
			"docker compose up --remove-orphans --wait --wait-timeout 600 -d",
		},
		DockerComposeDown: []string{"docker compose down"},
		DockerComposeRollout: []string{
			"docker compose pull --ignore-buildable --quiet || docker compose pull --ignore-buildable",
			"docker compose build --pull",
			"mkdir -p ./secrets",
			"docker compose run --rm init",
			"docker compose up --remove-orphans --pull missing --quiet-pull -d",
			"printf '%s\\n' 'ACTION REQUIRED: Omeka S may require a database migration. Sign in at /admin and complete any migration prompt before resuming normal operation.'",
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
	s.RegisterIngressRouteProvider(plugin.StandardComposeWebIngressRoutesWithOptions(plugin.StandardComposeWebIngressOptions{
		AppService: "omeka-s",
		Router:     "omeka-s-web",
	}))
	registerOmekaSCommands(s)
}

func registerApplicationComponents(s *plugin.SDK, displayName, appService string) {
	ingress, err := coretraefik.Ingress(coretraefik.IngressOptions{
		AppService:      appService,
		HTTPEntrypoint:  "web",
		HTTPSEntrypoint: "websecure",
		AppEnvDeletes:   []string{"DOMAIN", "OMEKA_S_ENABLE_HTTPS"},
	})
	if err != nil {
		panic(err)
	}
	s.RegisterServiceComponents(plugin.ServiceComponentRegistryOptions{
		DisplayName: displayName,
		Components:  []corecomponent.ComposeServiceComponent{ingress},
	})
}

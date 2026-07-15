package cmd

import (
	corecomponent "github.com/libops/sitectl/pkg/component"
	"github.com/libops/sitectl/pkg/plugin"
	coretraefik "github.com/libops/sitectl/pkg/services/traefik"
)

const (
	createRepo   = "https://github.com/libops/omeka-s"
	createBranch = "v1.0.0"
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
			"docker compose up --remove-orphans --pull missing --quiet-pull -d omeka-s",
			"docker compose exec -T omeka-s sh -c 'started=$(date +%s) || exit 1; deadline=$((started + 600)); until test -f /installed && curl --connect-timeout 2 --max-time 5 -fsS http://127.0.0.1/status | grep -q pool; do now=$(date +%s) || exit 1; if [ \"$now\" -ge \"$deadline\" ]; then echo \"Omeka S did not become ready for migration inspection within 10 minutes\" >&2; exit 1; fi; sleep 2; done'",
			"docker compose exec -T omeka-s sh -c 'result=$(curl --connect-timeout 2 --max-time 30 -sS -o /dev/null -w \"%{http_code} %{redirect_url}\" http://127.0.0.1/admin) || { status=$?; echo \"Unable to inspect Omeka S migration state (curl status $status)\" >&2; exit \"$status\"; }; code=${result%% *}; redirect=${result#* }; case \"$code\" in 200|301|302|303|307|308) ;; *) echo \"Unexpected Omeka S admin response: $code\" >&2; exit 3 ;; esac; case \"$redirect\" in */migrate|*/migrate/) printf \"%s\\n\" \"ACTION REQUIRED: Omeka S requires its supported browser migration. Public Traefik remains stopped. Run sitectl port-forward 8080:omeka-s:80, open http://localhost:8080/admin, complete the migration, stop the forward, and rerun sitectl deploy --skip-git --no-pull. If this deploy selected a non-active context, pass the same --context NAME to both sitectl commands.\" >&2; exit 10 ;; esac'",
			"docker compose up --remove-orphans --wait --wait-timeout 600 --pull missing --quiet-pull -d",
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

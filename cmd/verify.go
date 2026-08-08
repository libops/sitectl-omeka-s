package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/libops/sitectl/pkg/config"
	"github.com/libops/sitectl/pkg/docker"
	"github.com/libops/sitectl/pkg/plugin"
	sitevalidate "github.com/libops/sitectl/pkg/validate"
	"github.com/spf13/cobra"
)

const (
	omekaSService          = "omeka-s"
	omekaSRoot             = "/var/www/omeka-s"
	omekaSExpectedVersion  = "4.2.1"
	omekaSDatabaseProbe    = `. /usr/local/share/libops/database.sh; mapfile -d '' -t database < <(php -r '$config = parse_ini_file("config/database.ini", false, INI_SCANNER_RAW); foreach (["host", "port", "user", "password", "dbname"] as $key) { $value = $config[$key] ?? ""; if (!is_string($value) || $value === "") { fwrite(STDERR, "config/database.ini " . $key . " is empty\n"); exit(2); } fwrite(STDOUT, $value . "\0"); }'); if [ "${#database[@]}" -ne 5 ]; then printf '%s\n' 'could not read database credentials from config/database.ini' >&2; exit 2; fi; database_mariadb_with_password "${database[3]}" --host="${database[0]}" --port="${database[1]}" --user="${database[2]}" --database="${database[4]}" --batch --skip-column-names --execute="SELECT CURRENT_USER();"`
	omekaSReadOnlyStorage  = `test -r /var/www/omeka-s/files && test -w /var/www/omeka-s/files && printf '%s\n' 'storage writable'`
	omekaSStorageRoundTrip = `probe=/var/www/omeka-s/files/.sitectl-verify-$$; cleanup() { rm -f -- "$probe"; }; trap cleanup EXIT INT TERM; printf '%s' sitectl-verify >"$probe"; test "$(cat "$probe")" = sitectl-verify; cleanup; trap - EXIT INT TERM; printf '%s\n' 'storage round trip complete'`
)

type omekaSVerifyRuntime interface {
	ExecCapture(context.Context, []string) (string, error)
}

type dockerOmekaSVerifyRuntime struct {
	client    *docker.DockerClient
	container string
}

func (r dockerOmekaSVerifyRuntime) ExecCapture(ctx context.Context, argv []string) (string, error) {
	return docker.ExecCapture(ctx, r.client, r.container, omekaSRoot, argv)
}

type omekaSVerifyRunner struct {
	sdk        *plugin.SDK
	disposable bool
}

func (r *omekaSVerifyRunner) BindFlags(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&r.disposable, "disposable", false, "Write, read, and remove a probe file in Omeka S storage. Use only for a disposable CI site, never a retained site.")
}

func (r *omekaSVerifyRunner) Run(cmd *cobra.Command, _ *config.Context) ([]sitevalidate.Result, error) {
	if r.sdk == nil {
		return nil, fmt.Errorf("verifier SDK for Omeka S is not initialized")
	}
	verifyContext, err := r.sdk.GetContext()
	if err != nil {
		return nil, err
	}
	client, err := r.sdk.GetDockerClient()
	if err != nil {
		return nil, fmt.Errorf("connect to Docker for Omeka S verification: %w", err)
	}
	defer func() { _ = client.Close() }()
	container, err := client.GetContainerNameContext(cmd.Context(), verifyContext, omekaSService)
	if err != nil {
		return nil, fmt.Errorf("find running Omeka S container: %w", err)
	}
	return runOmekaSVerifyChecks(cmd.Context(), dockerOmekaSVerifyRuntime{client: client, container: container}, r.disposable), nil
}

func runOmekaSVerifyChecks(ctx context.Context, runtime omekaSVerifyRuntime, disposable bool) []sitevalidate.Result {
	results := make([]sitevalidate.Result, 0, 5)

	versionOutput, versionErr := runtime.ExecCapture(ctx, []string{"php", "-r", `require "vendor/autoload.php"; require "application/Module.php"; echo \Omeka\Module::VERSION;`})
	results = append(results, omekaSVersionResult(versionOutput, versionErr))

	databaseOutput, databaseErr := runtime.ExecCapture(ctx, []string{"bash", "-lc", omekaSDatabaseProbe})
	results = append(results, omekaSDatabaseResult(databaseOutput, databaseErr))

	migrationOutput, migrationErr := runtime.ExecCapture(ctx, []string{"curl", "--connect-timeout", "2", "--max-time", "30", "-sS", "-o", "/dev/null", "-w", "%{http_code} %{redirect_url}", "http://127.0.0.1/admin"})
	results = append(results, omekaSMigrationResult(migrationOutput, migrationErr))

	apiOutput, apiErr := runtime.ExecCapture(ctx, []string{"curl", "--connect-timeout", "2", "--max-time", "30", "-fsS", "-H", "Accept: application/json", "http://127.0.0.1/api/sites?per_page=1"})
	results = append(results, omekaSAPIResult(apiOutput, apiErr))

	storageScript := omekaSReadOnlyStorage
	storageDetail := "files storage is writable by the Omeka S service account"
	if disposable {
		storageScript = omekaSStorageRoundTrip
		storageDetail = "files storage completed a reversible write/read/delete round trip"
	}
	_, storageErr := runtime.ExecCapture(ctx, []string{"s6-setuidgid", "nginx", "sh", "-ec", storageScript})
	if storageErr != nil {
		results = append(results, omekaSVerifyFailed("verify:omeka-s:files", storageErr.Error(), "repair ownership and permissions for /var/www/omeka-s/files"))
	} else {
		results = append(results, omekaSVerifyOK("verify:omeka-s:files", storageDetail))
	}

	return results
}

func omekaSVersionResult(output string, commandErr error) sitevalidate.Result {
	if commandErr != nil {
		return omekaSVerifyFailed("verify:omeka-s:version", commandErr.Error(), "confirm the Omeka S application tree and Composer dependencies are complete")
	}
	version := strings.TrimSpace(output)
	if version != omekaSExpectedVersion {
		return omekaSVerifyFailed("verify:omeka-s:version", fmt.Sprintf("running version is %q, expected %s", version, omekaSExpectedVersion), "rebuild from the plugin's supported Omeka S base image")
	}
	return omekaSVerifyOK("verify:omeka-s:version", version)
}

func omekaSDatabaseResult(output string, commandErr error) sitevalidate.Result {
	if commandErr != nil {
		return omekaSVerifyFailed("verify:omeka-s:database-identity", commandErr.Error(), "check the scoped Omeka S database secret and MariaDB connectivity")
	}
	identity := strings.TrimSpace(output)
	if identity == "" {
		return omekaSVerifyFailed("verify:omeka-s:database-identity", "database returned no current user", "check the scoped Omeka S database secret")
	}
	username, _, _ := strings.Cut(identity, "@")
	if strings.EqualFold(username, "root") {
		return omekaSVerifyFailed("verify:omeka-s:database-identity", "Omeka S is connected as the MariaDB root user", "configure Omeka S with its scoped application database user")
	}
	return omekaSVerifyOK("verify:omeka-s:database-identity", identity)
}

func omekaSMigrationResult(output string, commandErr error) sitevalidate.Result {
	if commandErr != nil {
		return omekaSVerifyFailed("verify:omeka-s:migration", commandErr.Error(), "inspect the private Omeka S admin route before reopening ingress")
	}
	fields := strings.Fields(output)
	if len(fields) == 0 {
		return omekaSVerifyFailed("verify:omeka-s:migration", "migration probe returned no HTTP status", "inspect the private Omeka S admin route")
	}
	code, err := strconv.Atoi(fields[0])
	if err != nil {
		return omekaSVerifyFailed("verify:omeka-s:migration", fmt.Sprintf("invalid HTTP status %q", fields[0]), "inspect the private Omeka S admin route")
	}
	if code != 200 && code != 301 && code != 302 && code != 303 && code != 307 && code != 308 {
		return omekaSVerifyFailed("verify:omeka-s:migration", fmt.Sprintf("unexpected admin HTTP status %d", code), "inspect the private Omeka S admin route")
	}
	redirect := ""
	if len(fields) > 1 {
		redirect = fields[1]
	}
	if redirect != "" {
		parsed, parseErr := url.Parse(redirect)
		if parseErr != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.User != nil {
			return omekaSVerifyFailed("verify:omeka-s:migration", fmt.Sprintf("invalid admin redirect %q", redirect), "inspect the private Omeka S admin route")
		}
		redirectPath := strings.TrimRight(parsed.Path, "/")
		if redirectPath == "/migrate" || strings.HasSuffix(redirectPath, "/migrate") {
			return omekaSVerifyFailed("verify:omeka-s:migration", "browser migration is required and public ingress must remain stopped", "run sitectl port-forward 8080:omeka-s:80, finish /admin, then rerun the same deploy context")
		}
	}
	return omekaSVerifyOK("verify:omeka-s:migration", fmt.Sprintf("admin route returned %d without a migration redirect", code))
}

func omekaSAPIResult(output string, commandErr error) sitevalidate.Result {
	if commandErr != nil {
		return omekaSVerifyFailed("verify:omeka-s:api", commandErr.Error(), "confirm the Omeka S REST API is reachable")
	}
	var sites []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &sites); err != nil {
		return omekaSVerifyFailed("verify:omeka-s:api", fmt.Sprintf("decode sites response: %v", err), "inspect the Omeka S REST API response")
	}
	for index, site := range sites {
		if _, ok := site["o:id"]; !ok {
			return omekaSVerifyFailed("verify:omeka-s:api", fmt.Sprintf("site %d omitted o:id", index), "inspect the Omeka S REST API response")
		}
	}
	return omekaSVerifyOK("verify:omeka-s:api", fmt.Sprintf("sites resource returned %d record(s)", len(sites)))
}

func omekaSVerifyOK(name, detail string) sitevalidate.Result {
	return sitevalidate.Result{Name: name, Status: sitevalidate.StatusOK, Detail: detail}
}

func omekaSVerifyFailed(name, detail, fix string) sitevalidate.Result {
	return sitevalidate.Result{Name: name, Status: sitevalidate.StatusFailed, Detail: detail, FixHint: fix}
}

var _ plugin.VerifyRunner = (*omekaSVerifyRunner)(nil)

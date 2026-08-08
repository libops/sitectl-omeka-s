package cmd

import (
	"context"
	"errors"
	"strings"
	"testing"

	sitevalidate "github.com/libops/sitectl/pkg/validate"
)

type fakeOmekaSVerifyRuntime struct {
	run func([]string) (string, error)
}

func (f fakeOmekaSVerifyRuntime) ExecCapture(_ context.Context, argv []string) (string, error) {
	return f.run(argv)
}

func TestOmekaSVerifyChecksApplicationBehavior(t *testing.T) {
	t.Parallel()

	runtime := fakeOmekaSVerifyRuntime{run: func(argv []string) (string, error) {
		joined := strings.Join(argv, " ")
		switch {
		case strings.Contains(joined, "Omeka\\Module::VERSION"):
			return "4.2.1", nil
		case strings.Contains(joined, "SELECT CURRENT_USER()"):
			return "omeka_s@%", nil
		case strings.Contains(joined, "-w") && strings.Contains(joined, "/admin"):
			return "302 http://127.0.0.1/admin/login", nil
		case strings.Contains(joined, "/api/sites"):
			return `[{"o:id":1,"o:title":"Museum"}]`, nil
		case strings.Contains(joined, "test -w"):
			return "storage writable", nil
		default:
			return "", errors.New("unexpected command: " + joined)
		}
	}}

	results := runOmekaSVerifyChecks(context.Background(), runtime, false)
	assertAllOmekaSVerifyOK(t, results, 5)
}

func TestOmekaSVerifyFailsClosedOnMigrationRedirect(t *testing.T) {
	t.Parallel()

	for _, redirect := range []string{
		"302 http://127.0.0.1/migrate",
		"302 http://127.0.0.1/admin/migrate/",
	} {
		result := omekaSMigrationResult(redirect, nil)
		if result.Status != sitevalidate.StatusFailed || !strings.Contains(result.FixHint, "port-forward") {
			t.Fatalf("migration-required state %q was not failed with recovery guidance: %+v", redirect, result)
		}
	}
}

func TestOmekaSVerifyRejectsNonHTTPAdminRedirect(t *testing.T) {
	t.Parallel()

	result := omekaSMigrationResult("302 ftp://127.0.0.1/admin/login", nil)
	if result.Status != sitevalidate.StatusFailed {
		t.Fatalf("non-HTTP admin redirect was accepted: %+v", result)
	}
}

func TestOmekaSVerifyRejectsMalformedAPIResponse(t *testing.T) {
	t.Parallel()

	result := omekaSAPIResult(`{"error":"not a collection"}`, nil)
	if result.Status != sitevalidate.StatusFailed {
		t.Fatalf("malformed sites response was accepted: %+v", result)
	}
}

func TestOmekaSVerifyRejectsRootDatabaseIdentity(t *testing.T) {
	t.Parallel()

	result := omekaSDatabaseResult("root@localhost", nil)
	if result.Status != sitevalidate.StatusFailed {
		t.Fatalf("root database identity was accepted: %+v", result)
	}
}

func TestOmekaSVerifyDisposableModeUsesReversibleFilesProbe(t *testing.T) {
	t.Parallel()

	var storageCommand string
	runtime := fakeOmekaSVerifyRuntime{run: func(argv []string) (string, error) {
		joined := strings.Join(argv, " ")
		switch {
		case strings.Contains(joined, "Omeka\\Module::VERSION"):
			return "4.2.1", nil
		case strings.Contains(joined, "SELECT CURRENT_USER()"):
			return "omeka_s@%", nil
		case strings.Contains(joined, "-w") && strings.Contains(joined, "/admin"):
			return "302 http://127.0.0.1/admin/login", nil
		case strings.Contains(joined, "/api/sites"):
			return `[]`, nil
		case strings.Contains(joined, ".sitectl-verify"):
			storageCommand = joined
			return "storage round trip complete", nil
		default:
			return "", errors.New("unexpected command: " + joined)
		}
	}}

	results := runOmekaSVerifyChecks(context.Background(), runtime, true)
	assertAllOmekaSVerifyOK(t, results, 5)
	for _, required := range []string{"s6-setuidgid nginx", ".sitectl-verify", "trap", "rm -f"} {
		if !strings.Contains(storageCommand, required) {
			t.Fatalf("disposable files probe missing %q: %s", required, storageCommand)
		}
	}
}

func assertAllOmekaSVerifyOK(t *testing.T, results []sitevalidate.Result, want int) {
	t.Helper()
	if len(results) != want {
		t.Fatalf("verification results = %d, want %d: %+v", len(results), want, results)
	}
	for _, result := range results {
		if result.Status != sitevalidate.StatusOK {
			t.Fatalf("verification result is not OK: %+v", result)
		}
	}
}

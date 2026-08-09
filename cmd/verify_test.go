package cmd

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	sitevalidate "github.com/libops/sitectl/pkg/validate"
)

type fakeOmekaSVerifyRuntime struct {
	run   func([]string) (string, error)
	calls [][]string
}

func (f *fakeOmekaSVerifyRuntime) ExecCapture(_ context.Context, argv []string) (string, error) {
	f.calls = append(f.calls, append([]string(nil), argv...))
	return f.run(argv)
}

func TestOmekaSVerifyChecksApplicationBehavior(t *testing.T) {
	t.Parallel()

	runtime := &fakeOmekaSVerifyRuntime{run: func(argv []string) (string, error) {
		switch {
		case argv[0] == "test":
			return "", nil
		case reflect.DeepEqual(argv, []string{"php", omekaSVersionTarget}):
			return "4.2.1", nil
		case reflect.DeepEqual(argv, []string{omekaSVerifyDatabaseTarget}):
			return "omeka_s@%", nil
		case argv[0] == "curl" && strings.Contains(argv[len(argv)-1], "/admin"):
			return "302 http://127.0.0.1/admin/login", nil
		case argv[0] == "curl" && strings.Contains(argv[len(argv)-1], "/api/sites"):
			return `[{"o:id":1,"o:title":"Museum"}]`, nil
		case reflect.DeepEqual(argv, []string{"s6-setuidgid", "nginx", omekaSVerifyStorageTarget, omekaSStorageReadOnlyMode}):
			return "storage writable", nil
		default:
			return "", errors.New("unexpected command: " + strings.Join(argv, " "))
		}
	}}

	results := runOmekaSVerifyChecks(context.Background(), runtime, false)
	assertAllOmekaSVerifyOK(t, results, 5)
	if len(runtime.calls) != len(omekaSTemplatePrograms)+5 {
		t.Fatalf("verification calls = %d, want %d: %+v", len(runtime.calls), len(omekaSTemplatePrograms)+5, runtime.calls)
	}
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

func TestOmekaSVerifyUsesCheckedInDatabasePrograms(t *testing.T) {
	t.Parallel()

	programs := map[string]string{
		omekaSDatabaseConfigSource: omekaSDatabaseConfigTarget,
		omekaSVerifyDatabaseSource: omekaSVerifyDatabaseTarget,
		omekaSVerifySQLSource:      omekaSVerifySQLTarget,
	}
	if len(programs) != 3 {
		t.Fatalf("unexpected database program contract: %+v", programs)
	}
	for source, target := range programs {
		if !strings.HasPrefix(source, "scripts/") || !strings.HasPrefix(target, "/usr/local/") {
			t.Fatalf("database program is not a checked-in stable-path mapping: %s -> %s", source, target)
		}
	}
}

func TestOmekaSVerifyDisposableModeUsesVersionedStorageProgram(t *testing.T) {
	t.Parallel()

	var storageCommand []string
	runtime := &fakeOmekaSVerifyRuntime{run: func(argv []string) (string, error) {
		switch {
		case argv[0] == "test":
			return "", nil
		case reflect.DeepEqual(argv, []string{"php", omekaSVersionTarget}):
			return "4.2.1", nil
		case reflect.DeepEqual(argv, []string{omekaSVerifyDatabaseTarget}):
			return "omeka_s@%", nil
		case argv[0] == "curl" && strings.Contains(argv[len(argv)-1], "/admin"):
			return "302 http://127.0.0.1/admin/login", nil
		case argv[0] == "curl" && strings.Contains(argv[len(argv)-1], "/api/sites"):
			return `[]`, nil
		case len(argv) == 4 && argv[0] == "s6-setuidgid":
			storageCommand = append([]string(nil), argv...)
			return "storage round trip complete", nil
		default:
			return "", errors.New("unexpected command: " + strings.Join(argv, " "))
		}
	}}

	results := runOmekaSVerifyChecks(context.Background(), runtime, true)
	assertAllOmekaSVerifyOK(t, results, 5)
	want := []string{"s6-setuidgid", "nginx", omekaSVerifyStorageTarget, omekaSStorageDisposableMode}
	if !reflect.DeepEqual(storageCommand, want) {
		t.Fatalf("disposable storage command = %#v, want %#v", storageCommand, want)
	}
}

func TestOmekaSVerifyFailsClearlyForOlderTemplate(t *testing.T) {
	t.Parallel()

	runtime := &fakeOmekaSVerifyRuntime{run: func(argv []string) (string, error) {
		if argv[0] == "test" && argv[2] == omekaSDatabaseConfigTarget {
			return "", errors.New("missing")
		}
		return "", nil
	}}

	results := runOmekaSVerifyChecks(context.Background(), runtime, false)
	if len(results) != 1 || results[0].Name != "verify:omeka-s:template-programs" || results[0].Status != sitevalidate.StatusFailed {
		t.Fatalf("unexpected compatibility result: %+v", results)
	}
	for _, want := range []string{omekaSDatabaseConfigTarget, createRepo, omekaSTemplateVersion} {
		if !strings.Contains(results[0].Detail+" "+results[0].FixHint, want) {
			t.Fatalf("compatibility result omitted %q: %+v", want, results[0])
		}
	}
	if len(runtime.calls) != 4 {
		t.Fatalf("verification continued after missing program: %+v", runtime.calls)
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

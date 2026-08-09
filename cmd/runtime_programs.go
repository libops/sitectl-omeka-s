package cmd

const (
	omekaSTemplateVersion = "v1.2.1"

	omekaSRolloutPreflightSource  = "scripts/sitectl-rollout-preflight.sh"
	omekaSRolloutPreflightCommand = "bash " + omekaSRolloutPreflightSource
	omekaSComposeConfigCommand    = "docker compose config --quiet"

	omekaSRolloutReadinessSource = "scripts/omeka-s-rollout-readiness.sh"
	omekaSRolloutReadinessTarget = "/usr/local/bin/sitectl-omeka-s-rollout-readiness"
	omekaSMigrationGateSource    = "scripts/omeka-s-rollout-migration-gate.sh"
	omekaSMigrationGateTarget    = "/usr/local/bin/sitectl-omeka-s-rollout-migration-gate"

	omekaSVersionSource         = "scripts/omeka-s-version.php"
	omekaSVersionTarget         = "/usr/local/share/libops/sitectl-omeka-s-version.php"
	omekaSDatabaseConfigSource  = "scripts/omeka-s-database-config.php"
	omekaSDatabaseConfigTarget  = "/usr/local/share/libops/sitectl-omeka-s-database-config.php"
	omekaSVerifyDatabaseSource  = "scripts/omeka-s-verify-database.sh"
	omekaSVerifyDatabaseTarget  = "/usr/local/bin/sitectl-omeka-s-verify-database"
	omekaSVerifySQLSource       = "scripts/omeka-s-verify.sql"
	omekaSVerifySQLTarget       = "/usr/local/share/libops/sitectl-omeka-s-verify.sql"
	omekaSVerifyStorageSource   = "scripts/omeka-s-verify-storage.sh"
	omekaSVerifyStorageTarget   = "/usr/local/bin/sitectl-omeka-s-verify-storage"
	omekaSStorageReadOnlyMode   = "--read-only"
	omekaSStorageDisposableMode = "--disposable"
)

type omekaSTemplateProgram struct {
	source     string
	target     string
	executable bool
}

var omekaSTemplatePrograms = []omekaSTemplateProgram{
	{source: omekaSRolloutReadinessSource, target: omekaSRolloutReadinessTarget, executable: true},
	{source: omekaSMigrationGateSource, target: omekaSMigrationGateTarget, executable: true},
	{source: omekaSVersionSource, target: omekaSVersionTarget},
	{source: omekaSDatabaseConfigSource, target: omekaSDatabaseConfigTarget},
	{source: omekaSVerifyDatabaseSource, target: omekaSVerifyDatabaseTarget, executable: true},
	{source: omekaSVerifySQLSource, target: omekaSVerifySQLTarget},
	{source: omekaSVerifyStorageSource, target: omekaSVerifyStorageTarget, executable: true},
}

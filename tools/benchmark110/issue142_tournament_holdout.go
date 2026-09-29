package main

import "github.com/natsuki0413/commiter-cli/internal/planning"

// Labels are evaluation data and are never included in model requests.
func issue142TournamentHoldouts() []fixture {
	return []fixture{
		{name: "holdout_crossdir_semantic", language: planning.English, files: []fileSpec{
			{path: "src/auth.go", diff: "+func RotateSession() bool { return true }\n"},
			{path: "src/telemetry.go", diff: "+func EmitLatency() bool { return true }\n"},
			{path: "docs/security.md", diff: "+Document when a rotated login session expires.\n"},
			{path: "docs/operations.md", diff: "+Document the latency metric for request monitoring.\n"},
		}, reference: [][]string{{"F001", "F003"}, {"F002", "F004"}}},
		{name: "holdout_same_directory_pairs", language: planning.English, files: []fileSpec{
			{path: "src/cache.go", diff: "+func CacheTTL() int { return 45 }\n"},
			{path: "src/cache_test.go", diff: "+func TestCacheTTL(t *testing.T) { if CacheTTL()!=45 { t.Fatal(\"ttl\") } }\n"},
			{path: "src/billing.go", diff: "+func InvoiceTotal() int { return 20 }\n"},
			{path: "src/billing_test.go", diff: "+func TestInvoiceTotal(t *testing.T) { if InvoiceTotal()!=20 { t.Fatal(\"total\") } }\n"},
		}, reference: [][]string{{"F001", "F002"}, {"F003", "F004"}}},
		{name: "holdout_spurious_test_link", language: planning.English, files: []fileSpec{
			{path: "src/export.go", diff: "+func ExportRows() int { return 3 }\n"},
			{path: "src/export_test.go", diff: "+func TestOldExportCleanup(t *testing.T) { /* remove obsolete fixture */ }\n"},
			{path: "src/logging.go", diff: "+func LogLevel() int { return 2 }\n"},
		}, reference: [][]string{{"F001"}, {"F002"}, {"F003"}}},
		{name: "holdout_atomic_feature", language: planning.English, files: []fileSpec{
			{path: "src/migration.go", diff: "+func ApplyMigration() bool { return true }\n"},
			{path: "src/migration_test.go", diff: "+func TestApplyMigration(t *testing.T) { if !ApplyMigration() { t.Fatal(\"migration\") } }\n"},
			{path: "docs/migration.md", diff: "+Document how to apply the new migration.\n"},
		}, reference: [][]string{{"F001", "F002", "F003"}}},
	}
}

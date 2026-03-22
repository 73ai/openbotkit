package tui

import (
	"testing"

	"github.com/73ai/openbotkit/config"
	"github.com/73ai/openbotkit/settings"
)

func testSvc(cfg *config.Config) *settings.Service {
	creds := make(map[string]string)
	return settings.New(cfg,
		settings.WithSaveFn(func(*config.Config) error { return nil }),
		settings.WithStoreCred(func(ref, value string) error {
			creds[ref] = value
			return nil
		}),
		settings.WithLoadCred(func(ref string) (string, error) {
			return creds[ref], nil
		}),
	)
}

func TestCloneBackupConfigNil(t *testing.T) {
	clone := cloneBackupConfig(nil)
	if clone != nil {
		t.Error("cloning nil should return nil")
	}
}

func TestCloneBackupConfigDeepCopy(t *testing.T) {
	original := &config.BackupConfig{
		Enabled:     true,
		Destination: "r2",
		Schedule:    "6h",
		R2: &config.R2Config{
			Bucket:   "my-bucket",
			Endpoint: "https://endpoint",
		},
		GDrive: &config.GDriveConfig{
			FolderID: "folder123",
		},
	}

	clone := cloneBackupConfig(original)

	// Verify values copied.
	if clone.Destination != "r2" || clone.Schedule != "6h" || !clone.Enabled {
		t.Error("top-level fields not copied")
	}
	if clone.R2.Bucket != "my-bucket" {
		t.Error("R2 fields not copied")
	}
	if clone.GDrive.FolderID != "folder123" {
		t.Error("GDrive fields not copied")
	}

	// Verify deep copy — mutating clone shouldn't affect original.
	clone.Destination = "gdrive"
	clone.R2.Bucket = "other-bucket"
	clone.GDrive.FolderID = "other-folder"

	if original.Destination != "r2" {
		t.Error("mutating clone affected original destination")
	}
	if original.R2.Bucket != "my-bucket" {
		t.Error("mutating clone affected original R2")
	}
	if original.GDrive.FolderID != "folder123" {
		t.Error("mutating clone affected original GDrive")
	}
}

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		input string
		hours int
	}{
		{"6h", 6},
		{"12h", 12},
		{"24h", 24},
		{"", 0},
		{"invalid", 0},
	}
	for _, tt := range tests {
		d := parseSchedule(tt.input)
		gotHours := int(d.Hours())
		if gotHours != tt.hours {
			t.Errorf("parseSchedule(%q) = %d hours, want %d", tt.input, gotHours, tt.hours)
		}
	}
}

func TestBackupDest(t *testing.T) {
	if settings.BackupDest(config.Default()) != "" {
		t.Error("should return empty for nil backup")
	}

	cfg := config.Default()
	cfg.Backup = &config.BackupConfig{Destination: "r2"}
	if settings.BackupDest(cfg) != "r2" {
		t.Error("should return r2")
	}
}

func TestEnsureBackup(t *testing.T) {
	cfg := config.Default()
	if cfg.Backup != nil {
		t.Fatal("precondition: backup should be nil")
	}
	ensureBackup(cfg)
	if cfg.Backup == nil {
		t.Error("ensureBackup should create backup config")
	}
}

func TestRollbackRestoresSnapshot(t *testing.T) {
	cfg := config.Default()
	cfg.Backup = &config.BackupConfig{
		Enabled:     true,
		Destination: "r2",
		Schedule:    "12h",
	}
	svc := testSvc(cfg)
	m := newModel(svc)

	// Snapshot the config.
	m.wizardBackupSnapshot = cloneBackupConfig(cfg.Backup)

	// Mutate config (simulating wizard mid-flow).
	cfg.Backup.Destination = "gdrive"
	cfg.Backup.Enabled = false

	// Rollback.
	m, _ = m.rollbackBackup()

	if cfg.Backup.Destination != "r2" {
		t.Errorf("destination not reverted: got %q, want r2", cfg.Backup.Destination)
	}
	if !cfg.Backup.Enabled {
		t.Error("enabled not reverted: got false, want true")
	}
	if cfg.Backup.Schedule != "12h" {
		t.Errorf("schedule not reverted: got %q, want 12h", cfg.Backup.Schedule)
	}
	if m.state != stateBrowse {
		t.Error("state should be stateBrowse after rollback")
	}
	if m.wizardBackupSnapshot != nil {
		t.Error("snapshot should be cleared after rollback")
	}
}

func TestEnterBackupWizardSkipsDestWhenSet(t *testing.T) {
	cfg := config.Default()
	cfg.Backup = &config.BackupConfig{
		Destination: "r2",
		// R2 not configured — should skip dest picker, go to auth (R2 creds).
	}
	svc := testSvc(cfg)
	m := newModel(svc)

	m, _ = m.enterBackupWizard()

	if m.state != stateBackupCreds {
		t.Errorf("should go to stateBackupCreds when dest set but not authed, got state %d", m.state)
	}
	if m.wizardBackupSnapshot == nil {
		t.Error("snapshot should be set")
	}
}

func TestEnterBackupWizardShowsDestWhenEmpty(t *testing.T) {
	cfg := config.Default()
	svc := testSvc(cfg)
	m := newModel(svc)

	m, _ = m.enterBackupWizard()

	if m.state != stateBackupDest {
		t.Errorf("should go to stateBackupDest when no dest set, got state %d", m.state)
	}
}

func TestCommitDestinationSwapSetsConfig(t *testing.T) {
	cfg := config.Default()
	cfg.Backup = &config.BackupConfig{
		Enabled:     true,
		Destination: "r2",
		R2: &config.R2Config{
			Bucket: "b", Endpoint: "e",
			AccessKeyRef: "ak", SecretKeyRef: "sk",
		},
		GDrive: &config.GDriveConfig{FolderID: "f123"},
	}
	svc := testSvc(cfg)
	m := newModel(svc)
	m.wizardBackupSnapshot = cloneBackupConfig(cfg.Backup)

	m, _ = m.commitDestinationSwap("gdrive")

	if cfg.Backup.Destination != "gdrive" {
		t.Errorf("destination should be gdrive, got %q", cfg.Backup.Destination)
	}
	if !cfg.Backup.Enabled {
		t.Error("should stay enabled when swapping to authenticated dest")
	}
	if m.state != stateBrowse {
		t.Error("should return to stateBrowse")
	}
	if m.wizardBackupSnapshot != nil {
		t.Error("snapshot should be cleared on commit")
	}
}

func TestSaveBackupSetsDefaults(t *testing.T) {
	cfg := config.Default()
	cfg.Backup = &config.BackupConfig{
		Destination: "r2",
		R2: &config.R2Config{
			Bucket: "b", Endpoint: "e",
			AccessKeyRef: "ak", SecretKeyRef: "sk",
		},
	}
	svc := testSvc(cfg)
	m := newModel(svc)

	m, _ = m.saveBackup()

	if !cfg.Backup.Enabled {
		t.Error("saveBackup should set enabled=true")
	}
	if cfg.Backup.Schedule != "6h" {
		t.Errorf("saveBackup should default schedule to 6h, got %q", cfg.Backup.Schedule)
	}
}

func TestSaveBackupPreservesExistingSchedule(t *testing.T) {
	cfg := config.Default()
	cfg.Backup = &config.BackupConfig{
		Destination: "r2",
		Schedule:    "24h",
		R2: &config.R2Config{
			Bucket: "b", Endpoint: "e",
			AccessKeyRef: "ak", SecretKeyRef: "sk",
		},
	}
	svc := testSvc(cfg)
	m := newModel(svc)

	m, _ = m.saveBackup()

	if cfg.Backup.Schedule != "24h" {
		t.Errorf("saveBackup should preserve existing schedule, got %q", cfg.Backup.Schedule)
	}
}

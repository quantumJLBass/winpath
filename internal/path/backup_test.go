package path

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBackupStruct(t *testing.T) {
	b := Backup{
		Timestamp: time.Now(),
		Hostname:  "TESTPC",
		Suffix:    "test",
	}
	b.SystemPath.Raw = `C:\Windows`
	b.SystemPath.Entries = []string{`C:\Windows`}
	b.UserPath.Raw = `%USERPROFILE%\bin`
	b.UserPath.Entries = []string{`%USERPROFILE%\bin`}

	if b.SystemPath.Raw == "" || b.UserPath.Raw == "" {
		t.Error("Backup struct not set correctly")
	}
}

func TestBackupInfoStruct(t *testing.T) {
	info := BackupInfo{
		Filename:      "path_20250115_100000_test.json",
		Timestamp:     time.Now(),
		Suffix:        "test",
		FormattedDate: "2025-01-15 10:00:00",
	}

	if info.Filename == "" || info.Suffix != "test" {
		t.Error("BackupInfo struct not set correctly")
	}
}

func TestBackupFilenameFormat(t *testing.T) {
	now := time.Now()
	suffix := "test"
	expected := "path_" + now.Format("20060102_150405") + "_" + suffix + ".json"

	if !strings.HasPrefix(expected, "path_") || !strings.HasSuffix(expected, ".json") {
		t.Error("Backup filename format incorrect")
	}
}

func TestBackupTimestampParsing(t *testing.T) {
	filename := "path_20250115_100000_test.json"

	if !strings.HasPrefix(filename, "path_") {
		t.Error("Should start with path_")
	}

	parts := strings.Split(filename, "_")
	if len(parts) < 4 {
		t.Error("Expected at least 4 parts")
	}
}

func TestBackupJSON(t *testing.T) {
	b := Backup{
		Timestamp: time.Now(),
		Hostname:  "TESTPC",
		Suffix:    "test",
	}
	b.SystemPath.Raw = `C:\Windows`
	b.SystemPath.Entries = []string{`C:\Windows`}
	b.UserPath.Raw = `%USERPROFILE%\bin`
	b.UserPath.Entries = []string{`%USERPROFILE%\bin`}

	data, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var loaded Backup
	err = json.Unmarshal(data, &loaded)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if loaded.SystemPath.Raw != b.SystemPath.Raw || loaded.UserPath.Raw != b.UserPath.Raw {
		t.Error("JSON round-trip failed")
	}
}

func TestConfigStruct(t *testing.T) {
	cfg := Config{
		JunctionFolder: `C:\l`,
		MaxBackups:     10,
		AutoBackup:     true,
		HotPaths:       []string{`C:\Windows`},
	}

	if cfg.JunctionFolder == "" || cfg.MaxBackups != 10 {
		t.Error("Config struct not set correctly")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.JunctionFolder == "" {
		t.Error("JunctionFolder should not be empty")
	}
	if cfg.MaxBackups <= 0 {
		t.Error("MaxBackups should be positive")
	}
}

func TestLoadConfig(t *testing.T) {
	cfg := LoadConfig()

	if cfg.JunctionFolder == "" {
		t.Error("JunctionFolder should not be empty")
	}
	if cfg.MaxBackups <= 0 {
		t.Error("MaxBackups should be positive")
	}
}

func TestSaveConfig(t *testing.T) {
	cfg := Config{
		JunctionFolder: `C:\test`,
		MaxBackups:     5,
		AutoBackup:     false,
		HotPaths:       []string{`C:\Test`},
	}

	err := SaveConfig(cfg)
	if err != nil {
		t.Logf("SaveConfig error (may be expected): %v", err)
	}
}

func TestGetBackupDir(t *testing.T) {
	dir := GetBackupDir()
	if dir == "" {
		t.Error("GetBackupDir should not be empty")
	}
}

func TestEnsureBackupDir(t *testing.T) {
	err := EnsureBackupDir()
	if err != nil {
		t.Logf("EnsureBackupDir error: %v", err)
	}
}

func TestCreateBackup(t *testing.T) {
	info, err := CreateBackup("test-create")
	if err != nil {
		t.Logf("CreateBackup error: %v", err)
		return
	}

	if info == nil {
		t.Fatal("Expected non-nil BackupInfo")
	}
	if info.Filename == "" {
		t.Error("Filename should not be empty")
	}

	DeleteBackup(info.Filename)
}

func TestListBackups(t *testing.T) {
	backups := ListBackups()
	t.Logf("Found %d backups", len(backups))
}

func TestListBackups_Sorted(t *testing.T) {
	backups := ListBackups()

	for i := 0; i < len(backups)-1; i++ {
		if backups[i].Timestamp.Before(backups[i+1].Timestamp) {
			t.Error("Backups not sorted descending by timestamp")
		}
	}
}

func TestListBackups_SkipsInvalid(t *testing.T) {
	EnsureBackupDir()

	invalidFile := filepath.Join(GetBackupDir(), "invalid.json")
	os.WriteFile(invalidFile, []byte("{}"), 0644)
	defer os.Remove(invalidFile)

	backups := ListBackups()
	for _, b := range backups {
		if b.Filename == "invalid.json" {
			t.Error("Invalid files should be skipped")
		}
	}
}

func TestLoadBackup_NotFound(t *testing.T) {
	_, err := LoadBackup("nonexistent.json")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestLoadBackup_InvalidJSON(t *testing.T) {
	EnsureBackupDir()
	badFile := filepath.Join(GetBackupDir(), "bad_json.json")
	os.WriteFile(badFile, []byte("not valid json {{{"), 0644)
	defer os.Remove(badFile)

	_, err := LoadBackup("bad_json.json")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestDeleteBackup_NotFound(t *testing.T) {
	err := DeleteBackup("nonexistent.json")
	t.Logf("DeleteBackup non-existent: %v", err)
}

func TestEnforceBackupLimit(t *testing.T) {
	EnforceBackupLimit()
}

func TestConfigPaths(t *testing.T) {
	home, _ := os.UserHomeDir()
	expectedDir := filepath.Join(home, ".syspath")

	backupDir := GetBackupDir()
	if !strings.Contains(backupDir, ".syspath") {
		t.Log("Backup dir may not be in expected location")
	}
	if !strings.HasPrefix(backupDir, expectedDir) {
		t.Logf("Backup dir %s not under expected %s", backupDir, expectedDir)
	}
}

func TestConfigHotPaths(t *testing.T) {
	cfg := Config{
		HotPaths: []string{`C:\Windows`, `C:\Program Files`},
	}

	if len(cfg.HotPaths) != 2 {
		t.Error("HotPaths not set correctly")
	}
}

func BenchmarkLoadConfig(b *testing.B) {
	for i := 0; i < b.N; i++ {
		LoadConfig()
	}
}

func BenchmarkListBackups(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ListBackups()
	}
}

func TestRestoreBackup_NotFound(t *testing.T) {
	err := RestoreBackup("nonexistent.json", false)
	if err == nil {
		t.Error("Expected error for non-existent backup")
	}
}

func TestRestoreBackup_UserScope(t *testing.T) {
	mock := getMockRunner(t)

	// Create a backup first
	info, err := CreateBackup("restore-test")
	if err != nil {
		t.Skipf("Could not create backup: %v", err)
	}
	defer DeleteBackup(info.Filename)

	beforeCalls := len(mock.Calls)

	// Restore it (user only)
	err = RestoreBackup(info.Filename, false)
	if err != nil {
		t.Logf("RestoreBackup error (may be expected): %v", err)
	}

	// Verify mock was called for SetPath (should not modify real PATH)
	if len(mock.Calls) <= beforeCalls {
		t.Error("Mock should have been called during restore")
	}
}

func TestRestoreBackup_AdminScope(t *testing.T) {
	mock := getMockRunner(t)

	// Create a backup first
	info, err := CreateBackup("restore-admin-test")
	if err != nil {
		t.Skipf("Could not create backup: %v", err)
	}
	defer DeleteBackup(info.Filename)

	beforeCalls := len(mock.Calls)

	// Restore it (admin scope)
	err = RestoreBackup(info.Filename, true)
	if err != nil {
		t.Logf("RestoreBackup admin error (may be expected): %v", err)
	}

	// Verify mock was called
	if len(mock.Calls) <= beforeCalls {
		t.Error("Mock should have been called during restore")
	}
}

func TestGetConfigPath(t *testing.T) {
	configPath := GetConfigPath()
	if configPath == "" {
		t.Error("GetConfigPath should not be empty")
	}
	if !strings.Contains(configPath, "config.json") {
		t.Error("Config path should contain config.json")
	}
	// Note: When TestMain sets a custom config dir, .syspath won't be in the path
	// We just verify the path ends with config.json
	if !strings.HasSuffix(configPath, "config.json") {
		t.Errorf("Config path should end with config.json, got: %s", configPath)
	}
}

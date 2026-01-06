package path

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// configDir can be overridden for testing
var configDir string

// SetConfigDir sets a custom config directory (for testing)
func SetConfigDir(dir string) {
	configDir = dir
}

// getConfigDir returns the config directory
func getConfigDir() string {
	if configDir != "" {
		return configDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".syspath")
}

// Backup represents a saved PATH backup
type Backup struct {
	Timestamp  time.Time `json:"timestamp"`
	Hostname   string    `json:"hostname"`
	Suffix     string    `json:"suffix"`
	SystemPath struct {
		Raw     string   `json:"raw"`
		Entries []string `json:"entries"`
	} `json:"systemPath"`
	UserPath struct {
		Raw     string   `json:"raw"`
		Entries []string `json:"entries"`
	} `json:"userPath"`
}

// BackupInfo contains metadata about a backup file
type BackupInfo struct {
	Filename      string
	Timestamp     time.Time
	Suffix        string
	FormattedDate string
}

// Config stores application configuration
type Config struct {
	JunctionFolder string   `json:"junctionFolder"`
	MaxBackups     int      `json:"maxBackups"`
	AutoBackup     bool     `json:"autoBackup"`
	HotPaths       []string `json:"hotPaths"`
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		JunctionFolder: `C:\l`,
		MaxBackups:     10,
		AutoBackup:     true,
		HotPaths:       []string{},
	}
}

// GetBackupDir returns the backup directory path
func GetBackupDir() string {
	return filepath.Join(getConfigDir(), "backups")
}

// GetConfigPath returns the config file path
func GetConfigPath() string {
	return filepath.Join(getConfigDir(), "config.json")
}

// EnsureBackupDir creates the backup directory if it doesn't exist
func EnsureBackupDir() error {
	dir := GetBackupDir()
	return os.MkdirAll(dir, 0755)
}

// LoadConfig loads configuration from disk
func LoadConfig() Config {
	config := DefaultConfig()
	data, err := os.ReadFile(GetConfigPath())
	if err != nil {
		return config
	}
	_ = json.Unmarshal(data, &config) // Ignore error, return default on failure
	return config
}

// SaveConfig saves configuration to disk
func SaveConfig(config Config) error {
	dir := filepath.Dir(GetConfigPath())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(GetConfigPath(), data, 0644)
}

// CreateBackup creates a new backup with the given suffix
func CreateBackup(suffix string) (*BackupInfo, error) {
	if err := EnsureBackupDir(); err != nil {
		return nil, err
	}

	backup := Backup{
		Timestamp: time.Now(),
		Hostname:  GetHostname(),
		Suffix:    suffix,
	}

	// Get current paths
	sysPath, _ := GetPathRaw("System")
	usrPath, _ := GetPathRaw("User")

	backup.SystemPath.Raw = sysPath
	backup.SystemPath.Entries = ParsePath(sysPath)
	backup.UserPath.Raw = usrPath
	backup.UserPath.Entries = ParsePath(usrPath)

	// Generate filename
	filename := fmt.Sprintf("path_%s_%s.json",
		backup.Timestamp.Format("20060102_150405"),
		suffix)

	filepath := filepath.Join(GetBackupDir(), filename)

	// Write backup
	data, err := json.MarshalIndent(backup, "", "  ")
	if err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return nil, err
	}

	// Enforce backup limit
	EnforceBackupLimit()

	return &BackupInfo{
		Filename:      filename,
		Timestamp:     backup.Timestamp,
		Suffix:        suffix,
		FormattedDate: backup.Timestamp.Format("2006-01-02 15:04:05"),
	}, nil
}

// ListBackups returns all available backups, sorted by date descending
func ListBackups() []BackupInfo {
	dir := GetBackupDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []BackupInfo{}
	}

	backups := make([]BackupInfo, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Parse filename: path_YYYYMMDD_HHMMSS_suffix.json
		name := strings.TrimSuffix(entry.Name(), ".json")
		parts := strings.Split(name, "_")
		if len(parts) < 4 || parts[0] != "path" {
			continue
		}

		// Parse timestamp
		dateStr := parts[1] + "_" + parts[2]
		timestamp, err := time.Parse("20060102_150405", dateStr)
		if err != nil {
			continue
		}

		suffix := strings.Join(parts[3:], "_")

		backups = append(backups, BackupInfo{
			Filename:      entry.Name(),
			Timestamp:     timestamp,
			Suffix:        suffix,
			FormattedDate: timestamp.Format("2006-01-02 15:04:05"),
		})
	}

	// Sort by timestamp descending
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups
}

// LoadBackup loads a backup from disk
func LoadBackup(filename string) (*Backup, error) {
	filepath := filepath.Join(GetBackupDir(), filename)
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	var backup Backup
	if err := json.Unmarshal(data, &backup); err != nil {
		return nil, err
	}

	return &backup, nil
}

// DeleteBackup deletes a backup file
func DeleteBackup(filename string) error {
	filepath := filepath.Join(GetBackupDir(), filename)
	return os.Remove(filepath)
}

// EnforceBackupLimit removes old backups to stay under the limit
func EnforceBackupLimit() {
	config := LoadConfig()
	backups := ListBackups()

	if len(backups) <= config.MaxBackups {
		return
	}

	// Delete oldest backups
	for i := config.MaxBackups; i < len(backups); i++ {
		_ = DeleteBackup(backups[i].Filename) // Best effort cleanup
	}
}

// RestoreBackup restores PATH from a backup
func RestoreBackup(filename string, isAdmin bool) error {
	backup, err := LoadBackup(filename)
	if err != nil {
		return err
	}

	// Create a backup of current state first
	_, _ = CreateBackup("pre-restore") // Best effort, don't fail restore

	// Restore user PATH
	if backup.UserPath.Raw != "" {
		if err := SetPath(backup.UserPath.Raw, "User"); err != nil {
			return fmt.Errorf("failed to restore user PATH: %w", err)
		}
	}

	// Restore system PATH if admin
	if isAdmin && backup.SystemPath.Raw != "" {
		if err := SetPath(backup.SystemPath.Raw, "System"); err != nil {
			return fmt.Errorf("failed to restore system PATH: %w", err)
		}
	}

	BroadcastEnvChange()
	return nil
}

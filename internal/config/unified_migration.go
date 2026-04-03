package config

import (
	"fmt"
	"os"
)

// UnifiedMigrationResult describes the files affected by migration.
type UnifiedMigrationResult struct {
	UnifiedPath      string
	ClientBackupPath string
	ServerBackupPath string
}

// MigrateToUnifiedConfig merges client and server config files into a unified config.yaml.
func MigrateToUnifiedConfig(clientPath, serverPath string) (*UnifiedMigrationResult, error) {
	paths := GetDefaultPaths()
	if clientPath == "" {
		clientPath = paths.ClientConfig
	}
	if serverPath == "" {
		serverPath = paths.ServerConfig
	}

	clientCfg, clientExists, err := loadClientConfigIfExists(clientPath)
	if err != nil {
		return nil, err
	}

	serverCfg, serverExists, err := loadServerConfigIfExists(clientPath, serverPath)
	if err != nil {
		return nil, err
	}

	if !clientExists && !serverExists {
		return nil, fmt.Errorf("no config files found to migrate")
	}

	unified := UnifiedConfigFile{}
	if clientCfg != nil {
		unified.Client = *clientCfg
	}
	if serverCfg != nil {
		unified.Server = serverCfg.Server
		unified.Editors = serverCfg.Editors
		unified.Logging = serverCfg.Logging
	}

	result := &UnifiedMigrationResult{UnifiedPath: clientPath}

	if clientExists {
		backupPath, backupErr := backupFile(clientPath)
		if backupErr != nil {
			return nil, backupErr
		}
		result.ClientBackupPath = backupPath
	}

	if err := SaveUnifiedConfig(clientPath, &unified); err != nil {
		return nil, err
	}

	if serverExists && serverPath != clientPath {
		backupPath, backupErr := backupFile(serverPath)
		if backupErr != nil {
			return nil, backupErr
		}
		result.ServerBackupPath = backupPath
		if err := os.Remove(serverPath); err != nil {
			return nil, fmt.Errorf("failed to remove legacy server config: %w", err)
		}
	}

	return result, nil
}

func loadClientConfigIfExists(path string) (*ClientConfig, bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, false, nil
	}

	cfg, err := LoadClientConfig(path)
	if err != nil {
		return nil, false, fmt.Errorf("failed to load client config: %w", err)
	}

	return cfg, true, nil
}

func loadServerConfigIfExists(clientPath, serverPath string) (*ServerConfigFile, bool, error) {
	if _, err := os.Stat(serverPath); err == nil {
		cfg, loadErr := LoadServerConfig(serverPath)
		if loadErr != nil {
			return nil, false, fmt.Errorf("failed to load server config: %w", loadErr)
		}

		return cfg, true, nil
	}

	if _, err := os.Stat(clientPath); os.IsNotExist(err) {
		return nil, false, nil
	}

	data, err := os.ReadFile(clientPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to read config file: %w", err)
	}
	if !hasNestedServerConfig(data) {
		return nil, false, nil
	}

	cfg, err := LoadServerConfig(clientPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to load unified server config: %w", err)
	}

	return cfg, true, nil
}

func backupFile(path string) (string, error) {
	backupPath := path + ".bak"
	for i := 1; ; i++ {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		backupPath = fmt.Sprintf("%s.bak.%d", path, i)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read config for backup: %w", err)
	}
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	return backupPath, nil
}

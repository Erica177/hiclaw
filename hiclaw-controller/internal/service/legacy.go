package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/hiclaw/hiclaw-controller/internal/executor"
)

// LegacyCompat handles backward-compatible operations that only apply in
// embedded mode: Manager Agent openclaw.json manipulation, workers/teams/humans
// registry JSON files, and shell-based registry cleanup scripts.
//
// In incluster mode, construct with zero-value paths and nil Executor —
// Enabled() will return false and all methods become no-ops.
type LegacyCompat struct {
	ManagerConfigPath string          // embedded: ~/openclaw.json
	RegistryPath      string          // embedded: ~/workers-registry.json
	Executor          *executor.Shell // for shell-based registry cleanup
	MatrixDomain      string          // for building Matrix user IDs
}

// LegacyConfig holds configuration for constructing a LegacyCompat.
type LegacyConfig struct {
	ManagerConfigPath string
	RegistryPath      string
	Executor          *executor.Shell
	MatrixDomain      string
}

func NewLegacyCompat(cfg LegacyConfig) *LegacyCompat {
	return &LegacyCompat{
		ManagerConfigPath: cfg.ManagerConfigPath,
		RegistryPath:      cfg.RegistryPath,
		Executor:          cfg.Executor,
		MatrixDomain:      cfg.MatrixDomain,
	}
}

// Enabled reports whether any legacy operations are configured.
func (l *LegacyCompat) Enabled() bool {
	return l != nil && (l.ManagerConfigPath != "" || l.RegistryPath != "" || l.Executor != nil)
}

// MatrixUserID builds a full Matrix user ID from a localpart username.
func (l *LegacyCompat) MatrixUserID(name string) string {
	return fmt.Sprintf("@%s:%s", name, l.MatrixDomain)
}

// --- Manager Config ---

// UpdateManagerGroupAllowFrom adds or removes a worker Matrix ID from the Manager's
// openclaw.json groupAllowFrom list. No-op if ManagerConfigPath is empty.
func (l *LegacyCompat) UpdateManagerGroupAllowFrom(workerMatrixID string, add bool) error {
	if l == nil || l.ManagerConfigPath == "" {
		return nil
	}

	data, err := os.ReadFile(l.ManagerConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read manager config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parse manager config: %w", err)
	}

	channels, _ := config["channels"].(map[string]interface{})
	if channels == nil {
		return nil
	}
	matrixCfg, _ := channels["matrix"].(map[string]interface{})
	if matrixCfg == nil {
		return nil
	}

	allowList := extractStringSlice(matrixCfg["groupAllowFrom"])

	if add {
		for _, id := range allowList {
			if id == workerMatrixID {
				return nil
			}
		}
		allowList = append(allowList, workerMatrixID)
	} else {
		filtered := make([]string, 0, len(allowList))
		for _, id := range allowList {
			if id != workerMatrixID {
				filtered = append(filtered, id)
			}
		}
		allowList = filtered
	}

	matrixCfg["groupAllowFrom"] = allowList

	out, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manager config: %w", err)
	}
	return os.WriteFile(l.ManagerConfigPath, out, 0644)
}

func extractStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch arr := v.(type) {
	case []interface{}:
		result := make([]string, 0, len(arr))
		for _, item := range arr {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case []string:
		return arr
	}
	return nil
}

// --- Workers Registry ---

// WorkerRegistryEntry describes a worker entry in workers-registry.json.
type WorkerRegistryEntry struct {
	Name            string   `json:"-"`
	MatrixUserID    string   `json:"matrix_user_id"`
	RoomID          string   `json:"room_id"`
	Runtime         string   `json:"runtime"`
	Deployment      string   `json:"deployment"`
	Skills          []string `json:"skills"`
	Role            string   `json:"role"`
	TeamID          *string  `json:"team_id"`
	Image           *string  `json:"image"`
	CreatedAt       string   `json:"created_at,omitempty"`
	SkillsUpdatedAt string   `json:"skills_updated_at"`
}

type workersRegistry struct {
	Version   int                            `json:"version"`
	UpdatedAt string                         `json:"updated_at"`
	Workers   map[string]WorkerRegistryEntry `json:"workers"`
}

// UpdateWorkersRegistry upserts a worker entry in workers-registry.json.
// No-op if RegistryPath is empty.
func (l *LegacyCompat) UpdateWorkersRegistry(entry WorkerRegistryEntry) error {
	if l == nil || l.RegistryPath == "" {
		return nil
	}

	reg, err := loadWorkersRegistry(l.RegistryPath)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	existing, exists := reg.Workers[entry.Name]
	if exists && existing.CreatedAt != "" {
		entry.CreatedAt = existing.CreatedAt
	} else {
		entry.CreatedAt = now
	}
	entry.SkillsUpdatedAt = now
	reg.Workers[entry.Name] = entry
	reg.UpdatedAt = now

	return saveWorkersRegistry(l.RegistryPath, reg)
}

// RemoveFromWorkersRegistry removes a worker entry from workers-registry.json.
// No-op if RegistryPath is empty.
func (l *LegacyCompat) RemoveFromWorkersRegistry(workerName string) error {
	if l == nil || l.RegistryPath == "" {
		return nil
	}

	reg, err := loadWorkersRegistry(l.RegistryPath)
	if err != nil {
		return err
	}

	delete(reg.Workers, workerName)
	reg.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	return saveWorkersRegistry(l.RegistryPath, reg)
}

func loadWorkersRegistry(path string) (*workersRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &workersRegistry{
				Version: 1,
				Workers: make(map[string]WorkerRegistryEntry),
			}, nil
		}
		return nil, fmt.Errorf("read workers registry: %w", err)
	}

	var reg workersRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse workers registry: %w", err)
	}
	if reg.Workers == nil {
		reg.Workers = make(map[string]WorkerRegistryEntry)
	}
	return &reg, nil
}

func saveWorkersRegistry(path string, reg *workersRegistry) error {
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal workers registry: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// --- Shell-based registry cleanup ---

// RemoveTeamFromRegistry removes a team from the teams registry via shell script.
// No-op if Executor is nil.
func (l *LegacyCompat) RemoveTeamFromRegistry(ctx context.Context, teamName string) error {
	if l == nil || l.Executor == nil {
		return nil
	}
	_, err := l.Executor.RunSimple(ctx,
		"/opt/hiclaw/agent/skills/team-management/scripts/manage-teams-registry.sh",
		"--action", "remove", "--team-name", teamName,
	)
	return err
}

// RemoveHumanFromRegistry removes a human from the humans registry via shell script.
// No-op if Executor is nil.
func (l *LegacyCompat) RemoveHumanFromRegistry(ctx context.Context, humanName string) error {
	if l == nil || l.Executor == nil {
		return nil
	}
	_, err := l.Executor.RunSimple(ctx,
		"/opt/hiclaw/agent/skills/human-management/scripts/manage-humans-registry.sh",
		"--action", "remove", "--name", humanName,
	)
	return err
}

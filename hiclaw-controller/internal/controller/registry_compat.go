package controller

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// WorkerRegistryEntry describes a worker entry in workers-registry.json.
// This is a backward-compatibility measure for Manager Agent; Phase 3 removes it.
type WorkerRegistryEntry struct {
	Name         string   `json:"-"`
	MatrixUserID string   `json:"matrix_user_id"`
	RoomID       string   `json:"room_id"`
	Runtime      string   `json:"runtime"`
	Deployment   string   `json:"deployment"`
	Skills       []string `json:"skills"`
	Role         string   `json:"role"`
	TeamID       *string  `json:"team_id"`
	Image        *string  `json:"image"`
	CreatedAt    string   `json:"created_at,omitempty"`
	SkillsUpdatedAt string `json:"skills_updated_at"`
}

type workersRegistry struct {
	Version   int                            `json:"version"`
	UpdatedAt string                         `json:"updated_at"`
	Workers   map[string]WorkerRegistryEntry `json:"workers"`
}

// UpdateWorkersRegistry upserts a worker entry in workers-registry.json.
func UpdateWorkersRegistry(registryPath string, entry WorkerRegistryEntry) error {
	reg, err := loadWorkersRegistry(registryPath)
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

	return saveWorkersRegistry(registryPath, reg)
}

// RemoveFromWorkersRegistry removes a worker entry from workers-registry.json.
func RemoveFromWorkersRegistry(registryPath, workerName string) error {
	reg, err := loadWorkersRegistry(registryPath)
	if err != nil {
		return err
	}

	delete(reg.Workers, workerName)
	reg.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	return saveWorkersRegistry(registryPath, reg)
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

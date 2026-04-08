package controller

import (
	"encoding/json"
	"fmt"
	"os"
)

// UpdateManagerGroupAllowFrom adds or removes a worker Matrix ID from the Manager's
// openclaw.json groupAllowFrom list. This is embedded-mode only — in incluster mode,
// Manager gets config from OSS and this function should not be called.
func UpdateManagerGroupAllowFrom(configPath, workerMatrixID string, add bool) error {
	data, err := os.ReadFile(configPath)
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
	return os.WriteFile(configPath, out, 0644)
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

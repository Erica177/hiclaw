package service

import "github.com/hiclaw/hiclaw-controller/internal/config"

// WorkerEnvBuilder constructs environment variable maps for worker containers.
// Configuration defaults are injected at construction time rather than read
// from os.Getenv at call time, keeping the service layer test-friendly.
type WorkerEnvBuilder struct {
	defaults config.WorkerEnvDefaults
}

func NewWorkerEnvBuilder(defaults config.WorkerEnvDefaults) *WorkerEnvBuilder {
	return &WorkerEnvBuilder{defaults: defaults}
}

// Build returns the env map for a worker container, merging per-worker
// credentials with cluster-wide defaults.
func (b *WorkerEnvBuilder) Build(workerName string, prov *WorkerProvisionResult) map[string]string {
	env := map[string]string{
		"HICLAW_WORKER_NAME":         workerName,
		"HICLAW_WORKER_GATEWAY_KEY":  prov.GatewayKey,
		"HICLAW_WORKER_MATRIX_TOKEN": prov.MatrixToken,
		"HICLAW_FS_ACCESS_KEY":       workerName,
		"HICLAW_FS_SECRET_KEY":       prov.MinIOPassword,
		"OPENCLAW_DISABLE_BONJOUR":   "1",
		"OPENCLAW_MDNS_HOSTNAME":     "hiclaw-w-" + workerName,
		"HOME":                       "/root/hiclaw-fs/agents/" + workerName,
	}

	for k, v := range map[string]string{
		"HICLAW_MATRIX_DOMAIN":  b.defaults.MatrixDomain,
		"HICLAW_FS_ENDPOINT":    b.defaults.FSEndpoint,
		"HICLAW_MINIO_ENDPOINT": b.defaults.MinIOEndpoint,
		"HICLAW_MINIO_BUCKET":   b.defaults.MinIOBucket,
		"HICLAW_STORAGE_PREFIX": b.defaults.StoragePrefix,
		"HICLAW_CONTROLLER_URL": b.defaults.ControllerURL,
		"HICLAW_AI_GATEWAY_URL": b.defaults.AIGatewayURL,
		"HICLAW_MATRIX_URL":     b.defaults.MatrixURL,
	} {
		if v != "" {
			env[k] = v
		}
	}

	return env
}

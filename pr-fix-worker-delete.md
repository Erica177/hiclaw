# fix(controller): auto-recreate pod when deleted while CR is still Running

## Problem

When a Worker or Manager pod is deleted externally (e.g. `kubectl delete pod`, node eviction, OOM kill) while the CR status is still `Running`, the reconciler skips reconciliation because `observedGeneration == generation` and `phase == desired`. The pod is never recreated, leaving the CR in a stale `Running` state with no backing pod.

## Fix

In `reconcileDesiredState` for both `WorkerReconciler` and `ManagerReconciler`, when `phase == desired == "Running"`, check the backend for actual pod/container status. If the pod is missing, trigger recreation via the existing `ensureWorkerRunning` / `ensureManagerRunning` methods which already handle pod recreation.

Detection runs on the existing 5-minute `reconcileInterval`, so a deleted pod is recreated within 5 minutes.

## Files Changed

- `hiclaw-controller/internal/controller/worker_controller.go` — add `ensurePodExists`, call it from `reconcileDesiredState`
- `hiclaw-controller/internal/controller/manager_controller.go` — add `ensureManagerPodExists`, same pattern

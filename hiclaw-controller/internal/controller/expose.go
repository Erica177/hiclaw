package controller

import (
	"context"
	"fmt"

	v1beta1 "github.com/hiclaw/hiclaw-controller/api/v1beta1"
	"github.com/hiclaw/hiclaw-controller/internal/gateway"
)

// domainForExpose generates the auto domain name for a worker's exposed port.
func domainForExpose(workerName string, port int) string {
	return fmt.Sprintf("worker-%s-%d-local.hiclaw.io", workerName, port)
}

// containerDNSName returns the FQDN for a worker container that Higress can resolve.
// Worker containers are created with a network alias "{name}.local" on hiclaw-net,
// so Higress can resolve this as a DNS service source domain.
func containerDNSName(workerName string) string {
	return fmt.Sprintf("%s.local", workerName)
}

// ReconcileExpose compares desired expose ports with current status, creates new
// gateway resources for added ports, and removes resources for deleted ports.
// Returns the new ExposedPortStatus list.
func ReconcileExpose(ctx context.Context, gw gateway.Client, workerName string, desired []v1beta1.ExposePort, current []v1beta1.ExposedPortStatus) ([]v1beta1.ExposedPortStatus, error) {
	if gw == nil {
		return current, nil
	}

	desiredSet := make(map[int]v1beta1.ExposePort)
	for _, ep := range desired {
		desiredSet[ep.Port] = ep
	}
	currentSet := make(map[int]v1beta1.ExposedPortStatus)
	for _, ep := range current {
		currentSet[ep.Port] = ep
	}

	var result []v1beta1.ExposedPortStatus
	var firstErr error

	for _, ep := range desired {
		if _, exists := currentSet[ep.Port]; exists {
			result = append(result, currentSet[ep.Port])
			continue
		}

		domain := domainForExpose(workerName, ep.Port)
		err := gw.ExposePort(ctx, gateway.PortExposeRequest{
			WorkerName:  workerName,
			ServiceHost: containerDNSName(workerName),
			Port:        ep.Port,
			Domain:      domain,
		})
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("expose port %d: %w", ep.Port, err)
			}
			continue
		}

		result = append(result, v1beta1.ExposedPortStatus{
			Port:   ep.Port,
			Domain: domain,
		})
	}

	for _, ep := range current {
		if _, stillDesired := desiredSet[ep.Port]; stillDesired {
			continue
		}

		err := gw.UnexposePort(ctx, gateway.PortExposeRequest{
			WorkerName: workerName,
			Port:       ep.Port,
			Domain:     ep.Domain,
		})
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("unexpose port %d: %w", ep.Port, err)
			}
		}
	}

	return result, firstErr
}

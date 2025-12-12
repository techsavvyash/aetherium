package service

import (
	"context"
	"fmt"

	"github.com/aetherium/aetherium/services/core/pkg/network"
)

// ProxyService provides high-level proxy management operations
type ProxyService struct {
	networkManager *network.Manager
}

// NewProxyService creates a new proxy service
func NewProxyService(networkManager *network.Manager) *ProxyService {
	return &ProxyService{
		networkManager: networkManager,
	}
}

// UpdateGlobalWhitelist updates the global domain whitelist
func (ps *ProxyService) UpdateGlobalWhitelist(ctx context.Context, domains []string) error {
	if err := ps.networkManager.UpdateGlobalWhitelist(domains); err != nil {
		return fmt.Errorf("failed to update global whitelist: %w", err)
	}
	return nil
}

// RegisterVMDomains registers per-VM domain whitelist
func (ps *ProxyService) RegisterVMDomains(ctx context.Context, vmID, vmName string, domains []string) error {
	if err := ps.networkManager.RegisterVMWithProxy(vmID, vmName, domains); err != nil {
		return fmt.Errorf("failed to register VM domains: %w", err)
	}
	return nil
}

// UnregisterVM removes a VM from proxy whitelist
func (ps *ProxyService) UnregisterVM(ctx context.Context, vmID string) error {
	if err := ps.networkManager.UnregisterVMFromProxy(vmID); err != nil {
		return fmt.Errorf("failed to unregister VM: %w", err)
	}
	return nil
}

// GetProxyStats returns current proxy statistics
func (ps *ProxyService) GetProxyStats(ctx context.Context) (*network.ProxyStats, error) {
	stats, err := ps.networkManager.GetProxyStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get proxy stats: %w", err)
	}
	return stats, nil
}

// GetProxyHealth checks proxy health status
func (ps *ProxyService) GetProxyHealth(ctx context.Context) error {
	if err := ps.networkManager.GetProxyHealth(); err != nil {
		return fmt.Errorf("proxy health check failed: %w", err)
	}
	return nil
}

// GetBlockedRequests returns recently blocked requests
func (ps *ProxyService) GetBlockedRequests(ctx context.Context, limit int) ([]network.BlockedRequest, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	blocked, err := ps.networkManager.GetBlockedRequests(limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get blocked requests: %w", err)
	}
	return blocked, nil
}

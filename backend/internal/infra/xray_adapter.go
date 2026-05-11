package infra

import (
	"xray2wg/backend/internal/domain"
	xrayinfra "xray2wg/backend/internal/infrastructure/xrayinfra"
)

// XrayAdapter implements [service.XrayProcess] using [xrayinfra.Manager].
type XrayAdapter struct {
	M *xrayinfra.Manager
}

func (a *XrayAdapter) Start(tunnelID int, xrayPort int, fwmark int, localGatewayIP string, nodes []*domain.VlessNode, strategy domain.BalancingStrategy) error {
	return a.M.Start(tunnelID, xrayPort, fwmark, localGatewayIP, nodes, strategy)
}

func (a *XrayAdapter) Stop(tunnelID int) error {
	return a.M.Stop(tunnelID)
}

func (a *XrayAdapter) NodeHealth(tunnelID int) []domain.NodeHealthEntry {
	raw := a.M.NodeHealth(tunnelID)
	if raw == nil {
		return nil
	}
	out := make([]domain.NodeHealthEntry, len(raw))
	for i, e := range raw {
		out[i] = domain.NodeHealthEntry{Tag: e.Tag, Alive: e.Alive, DelayMs: e.DelayMs}
	}
	return out
}

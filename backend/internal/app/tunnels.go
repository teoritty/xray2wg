package app

import (
	"context"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/service"
)

// TunnelsApp coordinates tunnel metadata updates with runtime invariants.
type TunnelsApp struct {
	repo domain.TunnelRepository
	svc  *service.TunnelService
}

func NewTunnelsApp(repo domain.TunnelRepository, svc *service.TunnelService) *TunnelsApp {
	return &TunnelsApp{repo: repo, svc: svc}
}

// PrepareTunnelPUT validates and merges a PUT body over the existing row before persistence.
func (a *TunnelsApp) PrepareTunnelPUT(ctx context.Context, id int64, existing, incoming *domain.WgInterface) error {
	if a.svc.IsRunning(id) && incoming.Name != "" && incoming.Name != existing.Name {
		return domain.NewValidationError("cannot change tunnel name while running")
	}
	PreserveTunnelUpdateBindings(existing, incoming)
	PreserveTunnelLifecycleOnMetadataUpdate(existing, incoming)
	if existing.SubscriptionID != nil && incoming.SubscriptionID != nil &&
		*existing.SubscriptionID != *incoming.SubscriptionID && incoming.ActiveNodeID == nil {
		if err := a.repo.ClearActiveNodeID(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

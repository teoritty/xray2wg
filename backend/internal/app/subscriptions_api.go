package app

import (
	"context"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/service"
)

// SubscriptionsAPI is the use-case surface for /api/v1/subscriptions*.
type SubscriptionsAPI struct {
	SubRepo    domain.SubscriptionRepository
	Subs       *service.SubscriptionService
	NodeHealth *service.NodeHealthMonitor
}

func NewSubscriptionsAPI(repo domain.SubscriptionRepository, subs *service.SubscriptionService, nodeHealth *service.NodeHealthMonitor) *SubscriptionsAPI {
	return &SubscriptionsAPI{SubRepo: repo, Subs: subs, NodeHealth: nodeHealth}
}

// NodeHealthInfo mirrors the frontend NodeHealthInfo shape and is embedded in node listings so
// the tunnel creation form can render a ping badge before any tunnel exists (issue #6).
type NodeHealthInfo struct {
	Alive   bool  `json:"alive"`
	DelayMs int64 `json:"delay_ms"`
}

// VlessNodeWithHealth augments domain.VlessNode with the latest reachability probe. The
// embedded pointer means the JSON shape is identical to the bare node plus an extra Health
// field, so the frontend does not need to fork its VlessNode type.
type VlessNodeWithHealth struct {
	*domain.VlessNode
	Health *NodeHealthInfo `json:"Health"`
}

func (a *SubscriptionsAPI) List(ctx context.Context) ([]*domain.Subscription, error) {
	return a.SubRepo.List(ctx)
}

type SubscriptionCreateInput struct {
	Name            string
	URL             string
	RefreshInterval int64
}

func (a *SubscriptionsAPI) Add(ctx context.Context, in SubscriptionCreateInput) (*domain.Subscription, error) {
	return a.Subs.Add(ctx, in.Name, in.URL, in.RefreshInterval)
}

func (a *SubscriptionsAPI) AddManualVlessNode(ctx context.Context, vlessURI string) (*domain.VlessNode, error) {
	return a.Subs.AddManualVlessNode(ctx, vlessURI)
}

func (a *SubscriptionsAPI) UpdateManualVlessNode(ctx context.Context, nodeID int64, vlessURI string) (*domain.VlessNode, error) {
	return a.Subs.UpdateManualVlessNode(ctx, nodeID, vlessURI)
}

func (a *SubscriptionsAPI) DeleteManualVlessNode(ctx context.Context, nodeID int64) error {
	return a.Subs.DeleteManualVlessNode(ctx, nodeID)
}

func (a *SubscriptionsAPI) Get(ctx context.Context, id int64) (*domain.Subscription, error) {
	return a.SubRepo.GetByID(ctx, id)
}

func (a *SubscriptionsAPI) Update(ctx context.Context, su *domain.Subscription) error {
	return a.SubRepo.Update(ctx, su)
}

func (a *SubscriptionsAPI) Delete(ctx context.Context, id int64) error {
	a.Subs.StopRefreshLoop(id)
	return a.SubRepo.Delete(ctx, id)
}

func (a *SubscriptionsAPI) Refresh(ctx context.Context, id int64) error {
	return a.Subs.FetchAndUpdate(ctx, id)
}

func (a *SubscriptionsAPI) ListNodes(ctx context.Context, subID int64) ([]VlessNodeWithHealth, error) {
	nodes, err := a.SubRepo.ListNodes(ctx, subID)
	if err != nil {
		return nil, err
	}
	return a.enrichWithHealth(nodes), nil
}

func (a *SubscriptionsAPI) enrichWithHealth(nodes []*domain.VlessNode) []VlessNodeWithHealth {
	out := make([]VlessNodeWithHealth, 0, len(nodes))
	for _, n := range nodes {
		entry := VlessNodeWithHealth{VlessNode: n}
		if a.NodeHealth != nil {
			if snap, ok := a.NodeHealth.Get(n.ID); ok {
				entry.Health = &NodeHealthInfo{Alive: snap.Alive, DelayMs: snap.DelayMs}
			}
		}
		out = append(out, entry)
	}
	return out
}

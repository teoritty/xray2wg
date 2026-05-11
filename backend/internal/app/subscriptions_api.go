package app

import (
	"context"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/service"
)

// SubscriptionsAPI is the use-case surface for /api/v1/subscriptions*.
type SubscriptionsAPI struct {
	SubRepo domain.SubscriptionRepository
	Subs    *service.SubscriptionService
}

func NewSubscriptionsAPI(repo domain.SubscriptionRepository, subs *service.SubscriptionService) *SubscriptionsAPI {
	return &SubscriptionsAPI{SubRepo: repo, Subs: subs}
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

func (a *SubscriptionsAPI) ListNodes(ctx context.Context, subID int64) ([]*domain.VlessNode, error) {
	return a.SubRepo.ListNodes(ctx, subID)
}

package main

import (
	"context"
	"testing"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/service"
)

type spySubscriptionLooper struct {
	ids []int64
}

func (s *spySubscriptionLooper) StartRefreshLoop(ctx context.Context, id int64) {
	s.ids = append(s.ids, id)
}

func TestStartSubscriptionLoopsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := startSubscriptionLoops(ctx, []*domain.Subscription{{ID: 1, Name: "x", URL: "https://example.com"}}, nil)
	if err == nil {
		t.Fatal("expected ctx error")
	}
}

func TestStartSubscriptionLoopsNilService(t *testing.T) {
	if err := startSubscriptionLoops(context.Background(), []*domain.Subscription{{ID: 1}}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestStartSubscriptionLoopsEmptySlice(t *testing.T) {
	if err := startSubscriptionLoops(context.Background(), []*domain.Subscription{}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestStartSubscriptionLoopsStartsActiveWithURL(t *testing.T) {
	spy := &spySubscriptionLooper{}
	subs := []*domain.Subscription{
		{ID: 10, Name: "live", URL: "https://example.com/sub", Status: domain.SubStatusActive},
	}
	if err := startSubscriptionLoops(context.Background(), subs, spy); err != nil {
		t.Fatal(err)
	}
	if len(spy.ids) != 1 || spy.ids[0] != 10 {
		t.Fatalf("expected StartRefreshLoop for id 10, got %v", spy.ids)
	}
}

func TestStartSubscriptionLoopsSkipsManualAndInactive(t *testing.T) {
	ctx := context.Background()
	subSvc := service.NewSubscriptionService(&panicSubRepo{}, service.NewEventLog(1))
	subs := []*domain.Subscription{
		{ID: 1, Name: service.ManualSubscriptionName, URL: "https://a"},
		{ID: 2, Name: "in", URL: "https://b", Status: domain.SubStatusInactive},
		{ID: 3, Name: "no", URL: "  ", Status: domain.SubStatusActive},
	}
	if err := startSubscriptionLoops(ctx, subs, subSvc); err != nil {
		t.Fatal(err)
	}
}

type panicSubRepo struct{}

func (panicSubRepo) List(ctx context.Context) ([]*domain.Subscription, error) {
	panic("List should not run for skipped subs only test")
}

func (panicSubRepo) Create(ctx context.Context, s *domain.Subscription) error { panic("ni") }
func (panicSubRepo) GetByID(ctx context.Context, id int64) (*domain.Subscription, error) {
	panic("ni")
}
func (panicSubRepo) Update(ctx context.Context, s *domain.Subscription) error { panic("ni") }
func (panicSubRepo) Delete(ctx context.Context, id int64) error { panic("ni") }
func (panicSubRepo) DeleteNodes(ctx context.Context, subscriptionID int64) error { panic("ni") }
func (panicSubRepo) SnapshotActiveNodesForSubscription(ctx context.Context, subscriptionID int64) ([]domain.ActiveNodeBinding, error) {
	panic("ni")
}
func (panicSubRepo) RemapActiveNodesAfterRefresh(ctx context.Context, subscriptionID int64, bindings []domain.ActiveNodeBinding, newNodes []*domain.VlessNode) error {
	panic("ni")
}
func (panicSubRepo) InsertNodes(ctx context.Context, nodes []*domain.VlessNode) error { panic("ni") }
func (panicSubRepo) ListNodes(ctx context.Context, subscriptionID int64) ([]*domain.VlessNode, error) {
	panic("ni")
}
func (panicSubRepo) ListAllNodes(ctx context.Context) ([]*domain.VlessNode, error) { panic("ni") }
func (panicSubRepo) GetNode(ctx context.Context, id int64) (*domain.VlessNode, error) { panic("ni") }
func (panicSubRepo) FindTunnelIDsUsingNode(ctx context.Context, nodeID int64) ([]int64, error) {
	panic("ni")
}
func (panicSubRepo) UpdateNode(ctx context.Context, n *domain.VlessNode) error { panic("ni") }
func (panicSubRepo) DeleteNode(ctx context.Context, id int64) error { panic("ni") }

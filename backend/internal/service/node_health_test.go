package service

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"xray2wg/backend/internal/domain"
)

type fakeSubRepo struct {
	nodes []*domain.VlessNode
}

func (f *fakeSubRepo) Create(context.Context, *domain.Subscription) error { return nil }
func (f *fakeSubRepo) GetByID(context.Context, int64) (*domain.Subscription, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeSubRepo) List(context.Context) ([]*domain.Subscription, error) { return nil, nil }
func (f *fakeSubRepo) Update(context.Context, *domain.Subscription) error   { return nil }
func (f *fakeSubRepo) Delete(context.Context, int64) error                  { return nil }
func (f *fakeSubRepo) DeleteNodes(context.Context, int64) error             { return nil }
func (f *fakeSubRepo) SnapshotActiveNodesForSubscription(context.Context, int64) ([]domain.ActiveNodeBinding, error) {
	return nil, nil
}
func (f *fakeSubRepo) RemapActiveNodesAfterRefresh(context.Context, int64, []domain.ActiveNodeBinding, []*domain.VlessNode) error {
	return nil
}
func (f *fakeSubRepo) InsertNodes(context.Context, []*domain.VlessNode) error { return nil }
func (f *fakeSubRepo) ListNodes(context.Context, int64) ([]*domain.VlessNode, error) {
	return nil, nil
}
func (f *fakeSubRepo) ListAllNodes(context.Context) ([]*domain.VlessNode, error) {
	return f.nodes, nil
}
func (f *fakeSubRepo) GetNode(context.Context, int64) (*domain.VlessNode, error) {
	return nil, domain.ErrNotFound
}
func (f *fakeSubRepo) FindTunnelIDsUsingNode(context.Context, int64) ([]int64, error) {
	return nil, nil
}
func (f *fakeSubRepo) UpdateNode(context.Context, *domain.VlessNode) error { return nil }
func (f *fakeSubRepo) DeleteNode(context.Context, int64) error            { return nil }

func TestNodeHealthMonitor_aliveAndDeadProbes(t *testing.T) {
	repo := &fakeSubRepo{nodes: []*domain.VlessNode{
		{ID: 1, Address: "alive.example", Port: 443},
		{ID: 2, Address: "dead.example", Port: 443},
	}}
	m := NewNodeHealthMonitor(repo, time.Hour, 100*time.Millisecond, 4)
	m.dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		if addr == "alive.example:443" {
			c1, c2 := net.Pipe()
			go func() { _ = c2.Close() }()
			return c1, nil
		}
		return nil, errors.New("connection refused")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	m.probeAll(ctx)

	got1, ok := m.Get(1)
	if !ok || !got1.Alive {
		t.Fatalf("alive node: snapshot=%+v ok=%v", got1, ok)
	}
	got2, ok := m.Get(2)
	if !ok || got2.Alive || got2.DelayMs != -1 {
		t.Fatalf("dead node: snapshot=%+v ok=%v", got2, ok)
	}
}

func TestNodeHealthMonitor_evictsRemovedNodes(t *testing.T) {
	repo := &fakeSubRepo{nodes: []*domain.VlessNode{
		{ID: 1, Address: "alive.example", Port: 443},
		{ID: 2, Address: "alive.example", Port: 443},
	}}
	m := NewNodeHealthMonitor(repo, time.Hour, 100*time.Millisecond, 4)
	m.dial = func(ctx context.Context, network, addr string) (net.Conn, error) {
		c1, c2 := net.Pipe()
		go func() { _ = c2.Close() }()
		return c1, nil
	}
	ctx := context.Background()
	m.probeAll(ctx)
	if _, ok := m.Get(2); !ok {
		t.Fatal("node 2 should be cached after first probe")
	}

	// Remove node 2 and re-probe.
	repo.nodes = []*domain.VlessNode{{ID: 1, Address: "alive.example", Port: 443}}
	m.probeAll(ctx)
	if _, ok := m.Get(2); ok {
		t.Fatal("node 2 should have been evicted from cache after disappearing from repo")
	}
	if _, ok := m.Get(1); !ok {
		t.Fatal("node 1 should still be in cache")
	}
}

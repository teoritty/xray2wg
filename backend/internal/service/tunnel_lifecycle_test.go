package service

import (
	"context"
	"testing"

	"xray2wg/backend/internal/domain"
	xrayinfra "xray2wg/backend/internal/infrastructure/xrayinfra"
	wginfra "xray2wg/backend/internal/infrastructure/wireguard"
	"xray2wg/backend/internal/infra"
)

func TestTunnelService_ShutdownAllPreservesPersistedRunningStatus(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryTunnelRepo(map[int64]*domain.WgInterface{
		1: {
			ID:        1,
			Name:      "running",
			TunName:   "tun-wg1",
			WgAddress: "10.100.1.1/24",
			Status:    domain.WgStatusRunning,
		},
	})
	svc := NewTunnelService(repo, nil, nil, &infra.XrayAdapter{M: xrayinfra.NewManager()}, &infra.WgAdapter{M: wginfra.NewManager()}, nil, NewEventLog(1), nil)
	svc.running.Mark(1)

	svc.ShutdownAll(ctx)

	if svc.IsRunning(1) {
		t.Fatal("expected graceful shutdown to clear runtime running state")
	}
	if got := repo.mustGet(t, 1).Status; got != domain.WgStatusRunning {
		t.Fatalf("persisted status = %q, want %q", got, domain.WgStatusRunning)
	}
}

func TestTunnelService_StopPersistsStoppedStatus(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryTunnelRepo(map[int64]*domain.WgInterface{
		1: {
			ID:        1,
			Name:      "running",
			TunName:   "tun-wg1",
			WgAddress: "10.100.1.1/24",
			Status:    domain.WgStatusRunning,
		},
	})
	svc := NewTunnelService(repo, nil, nil, &infra.XrayAdapter{M: xrayinfra.NewManager()}, &infra.WgAdapter{M: wginfra.NewManager()}, nil, NewEventLog(1), nil)
	svc.running.Mark(1)

	if err := svc.Stop(ctx, 1); err != nil {
		t.Fatal(err)
	}

	if svc.IsRunning(1) {
		t.Fatal("expected explicit stop to clear runtime running state")
	}
	if got := repo.mustGet(t, 1).Status; got != domain.WgStatusStopped {
		t.Fatalf("persisted status = %q, want %q", got, domain.WgStatusStopped)
	}
}

type memoryTunnelRepo struct {
	tunnels map[int64]*domain.WgInterface
}

func newMemoryTunnelRepo(tunnels map[int64]*domain.WgInterface) *memoryTunnelRepo {
	return &memoryTunnelRepo{tunnels: tunnels}
}

func (r *memoryTunnelRepo) mustGet(t *testing.T, id int64) *domain.WgInterface {
	t.Helper()
	iface, _, err := r.GetByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return iface
}

func (r *memoryTunnelRepo) Create(ctx context.Context, iface *domain.WgInterface, privKeyEnc string) error {
	r.tunnels[iface.ID] = cloneTunnel(iface)
	return nil
}

func (r *memoryTunnelRepo) GetByID(ctx context.Context, id int64) (*domain.WgInterface, string, error) {
	iface, ok := r.tunnels[id]
	if !ok {
		return nil, "", domain.ErrNotFound
	}
	return cloneTunnel(iface), "", nil
}

func (r *memoryTunnelRepo) List(ctx context.Context) ([]*domain.WgInterface, error) {
	out := make([]*domain.WgInterface, 0, len(r.tunnels))
	for _, iface := range r.tunnels {
		out = append(out, cloneTunnel(iface))
	}
	return out, nil
}

func (r *memoryTunnelRepo) Update(ctx context.Context, iface *domain.WgInterface) error {
	if _, ok := r.tunnels[iface.ID]; !ok {
		return domain.ErrNotFound
	}
	r.tunnels[iface.ID] = cloneTunnel(iface)
	return nil
}

func (r *memoryTunnelRepo) ClearActiveNodeID(ctx context.Context, id int64) error {
	iface, ok := r.tunnels[id]
	if !ok {
		return domain.ErrNotFound
	}
	iface.ActiveNodeID = nil
	return nil
}

func (r *memoryTunnelRepo) Delete(ctx context.Context, id int64) error {
	if _, ok := r.tunnels[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.tunnels, id)
	return nil
}

func (r *memoryTunnelRepo) UpdateStatus(ctx context.Context, id int64, status domain.WgStatus, errMsg string) error {
	iface, ok := r.tunnels[id]
	if !ok {
		return domain.ErrNotFound
	}
	iface.Status = status
	iface.ErrorMessage = errMsg
	return nil
}

func (r *memoryTunnelRepo) UpdateRuntimeFields(ctx context.Context, id int64, tunName string, xrayPort, fwmark int) error {
	iface, ok := r.tunnels[id]
	if !ok {
		return domain.ErrNotFound
	}
	iface.TunName = tunName
	iface.XrayPort = xrayPort
	iface.FWMark = fwmark
	return nil
}

func (r *memoryTunnelRepo) ListRunningIDs(ctx context.Context) ([]int64, error) {
	var ids []int64
	for id, iface := range r.tunnels {
		if iface.Status == domain.WgStatusRunning {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (r *memoryTunnelRepo) CountPeers(ctx context.Context, interfaceID int64) (int64, error) {
	return 0, nil
}

func (r *memoryTunnelRepo) SetNodes(_ context.Context, _ int64, _ []int64) error { return nil }

func (r *memoryTunnelRepo) ListNodes(_ context.Context, _ int64) ([]*domain.VlessNode, error) {
	return nil, nil
}

func cloneTunnel(iface *domain.WgInterface) *domain.WgInterface {
	if iface == nil {
		return nil
	}
	cp := *iface
	return &cp
}

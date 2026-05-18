package service

import (
	"context"
	"testing"
	"time"

	"xray2wg/backend/internal/domain"
)

type stubPeerRepo struct {
	peers []*domain.WgPeer
}

func (s stubPeerRepo) Create(context.Context, int64, *domain.WgPeer, *string, *string) error {
	return nil
}
func (s stubPeerRepo) GetByID(context.Context, int64, int64) (*domain.WgPeer, string, string, error) {
	return nil, "", "", domain.ErrNotFound
}
func (s stubPeerRepo) ListByInterface(context.Context, int64) ([]*domain.WgPeer, error) {
	return s.peers, nil
}
func (s stubPeerRepo) ListAllWithTunnel(context.Context) ([]*domain.PeerWithTunnel, error) {
	return nil, nil
}
func (s stubPeerRepo) Update(context.Context, *domain.WgPeer, *string, *string) error {
	return nil
}
func (s stubPeerRepo) Delete(context.Context, int64, int64) error { return nil }
func (s stubPeerRepo) UpdateTraffic(context.Context, int64, *time.Time, int64, int64) (int64, int64, error) {
	return 0, 0, nil
}
func (s stubPeerRepo) GetByPubKey(context.Context, int64, string) (*domain.WgPeer, error) {
	return nil, domain.ErrNotFound
}

func TestNextClientIP_firstFree(t *testing.T) {
	ps := PeerService{
		repo: stubPeerRepo{peers: nil},
	}
	ip, err := ps.NextClientIP(context.Background(), 1, "10.100.3.0/24")
	if err != nil || ip != "10.100.3.2/32" {
		t.Fatalf("got %q %v", ip, err)
	}
}

func TestNextClientIP_skips_used(t *testing.T) {
	ps := PeerService{
		repo: stubPeerRepo{peers: []*domain.WgPeer{
			{ClientAddress: "10.100.9.2/32"},
			{ClientAddress: "10.100.9.3/32"},
		}},
	}
	ip, err := ps.NextClientIP(context.Background(), 1, "10.100.9.0/24")
	if err != nil || ip != "10.100.9.4/32" {
		t.Fatalf("got %q %v", ip, err)
	}
}

package app

import (
	"context"

	"xray2wg/backend/internal/domain"
)

// PeersAPI is the use-case surface for /api/v1/peers.
type PeersAPI struct {
	PeerRepo domain.PeerRepository
}

func NewPeersAPI(repo domain.PeerRepository) *PeersAPI {
	return &PeersAPI{PeerRepo: repo}
}

func (a *PeersAPI) ListAllWithTunnel(ctx context.Context) ([]*domain.PeerWithTunnel, error) {
	return a.PeerRepo.ListAllWithTunnel(ctx)
}

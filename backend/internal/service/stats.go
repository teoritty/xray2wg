package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"xray2wg/backend/internal/ctxlog"
	"xray2wg/backend/internal/domain"
	wginfra "xray2wg/backend/internal/infrastructure/wireguard"

	wshub "xray2wg/backend/internal/ws"
)

type StatsCollector struct {
	repo       domain.StatsRepository
	peers      domain.PeerRepository
	tunnels    domain.TunnelRepository
	hub        *wshub.Hub

	mu       sync.Mutex
	tracking map[int64]struct{}
	lastTot  map[int64][2]int64 // iface -> last sum rx tx for rate
	lastSeen map[int64][2]int64 // peer id sum
}

func NewStatsCollector(sr domain.StatsRepository, pr domain.PeerRepository, tr domain.TunnelRepository, hub *wshub.Hub) *StatsCollector {
	return &StatsCollector{
		repo: sr, peers: pr, tunnels: tr, hub: hub,
		tracking: make(map[int64]struct{}),
		lastTot:  make(map[int64][2]int64),
		lastSeen: make(map[int64][2]int64),
	}
}

func (s *StatsCollector) Track(iface int64) {
	s.mu.Lock()
	s.tracking[iface] = struct{}{}
	s.mu.Unlock()
}

func (s *StatsCollector) Untrack(iface int64) {
	s.mu.Lock()
	delete(s.tracking, iface)
	s.mu.Unlock()
}

func (s *StatsCollector) Run(ctx context.Context) {
	go s.cleanupLoop(ctx)
	t := time.NewTicker(2 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *StatsCollector) cleanupLoop(ctx context.Context) {
	tc := time.NewTicker(10 * time.Minute)
	defer tc.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tc.C:
			cut := time.Now().Add(-24 * time.Hour)
			_ = s.repo.DeleteOlderThan(ctx, cut)
		}
	}
}

func (s *StatsCollector) tick(ctx context.Context) {
	s.mu.Lock()
	ids := make([]int64, 0, len(s.tracking))
	for id := range s.tracking {
		ids = append(ids, id)
	}
	s.mu.Unlock()

	var tunnels []wshub.TunnelStats
	for _, id := range ids {
		iface, _, err := s.tunnels.GetByID(ctx, id)
		if err != nil {
			ctxlog.From(ctx).Debug().Int64("tunnel_id", id).Err(err).Msg("stats tick: GetByID failed, skipping")
			continue
		}
		tun := iface.TunName
		if tun == "" {
			tun = fmt.Sprintf("tun-wg%d", id)
		}
		ps, err := wginfra.PollStats(ctx, tun)
		if err != nil {
			// Most likely cause: wgctrl can't reach the userspace device's
			// UAPI socket at /var/run/wireguard/<tun>.sock. Surface it so the
			// frozen-stats symptom is diagnosable from logs.
			ctxlog.From(ctx).Warn().Int64("tunnel_id", id).Str("tun", tun).Err(err).Msg("stats tick: PollStats failed, no live rates this round")
			continue
		}
		var sumRx, sumTx int64
		active := 0
		now := time.Now().UTC()
		for _, st := range ps {
			pub := strings.TrimSpace(st.PublicKey)
			p, err := s.peers.GetByPubKey(ctx, id, pub)
			if err != nil || p == nil {
				continue
			}
			var lhs *time.Time
			if !st.LastHandshake.IsZero() {
				t := st.LastHandshake
				lhs = &t
			}
			accumRx, accumTx, err := s.peers.UpdateTraffic(ctx, p.ID, lhs, st.RxBytes, st.TxBytes)
			if err != nil {
				ctxlog.From(ctx).Warn().Int64("peer_id", p.ID).Err(err).Msg("stats tick: UpdateTraffic failed")
				continue
			}
			sumRx += accumRx
			sumTx += accumTx
			if !st.LastHandshake.IsZero() && now.Sub(st.LastHandshake) < 3*time.Minute {
				active++
			}
			prev := s.lastSeen[p.ID]
			dt := 2.0
			rxR := int64(float64(accumRx-prev[0]) / dt)
			txR := int64(float64(accumTx-prev[1]) / dt)
			if rxR < 0 {
				rxR = 0
			}
			if txR < 0 {
				txR = 0
			}
			s.lastSeen[p.ID] = [2]int64{accumRx, accumTx}
			pid := p.ID
			_ = s.repo.Insert(ctx, &domain.StatSnapshot{
				PeerID: &pid, RxBytes: accumRx, TxBytes: accumTx, RxRate: rxR, TxRate: txR, SampledAt: now,
			})
		}

		pt := int32(active)
		prevT := s.lastTot[id]
		rxRT := int64(float64(sumRx-prevT[0]) / 2.0)
		txRT := int64(float64(sumTx-prevT[1]) / 2.0)
		s.lastTot[id] = [2]int64{sumRx, sumTx}
		iid := id
		_ = s.repo.Insert(ctx, &domain.StatSnapshot{
			InterfaceID: &iid,
			RxBytes: sumRx, TxBytes: sumTx, RxRate: rxRT, TxRate: txRT,
			ActivePeers: &pt, SampledAt: now,
		})

		tunnels = append(tunnels, wshub.TunnelStats{
			ID: id, RxRate: rxRT, TxRate: txRT,
		})
	}
	if s.hub != nil {
		s.hub.Broadcast(wshub.Message{Tunnels: tunnels, Timestamp: time.Now().UnixMilli()})
	}
}

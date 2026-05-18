package service

import (
	"context"
	"maps"
	"net"
	"strconv"
	"sync"
	"time"

	"xray2wg/backend/internal/ctxlog"
	"xray2wg/backend/internal/domain"
)

// NodeHealthSnapshot is a point-in-time TCP reachability probe of a VLESS node.
// DelayMs is the TCP handshake latency in milliseconds; -1 when the node is unreachable.
type NodeHealthSnapshot struct {
	Alive     bool
	DelayMs   int64
	SampledAt time.Time
}

// NodeHealthMonitor probes every known VLESS node periodically and caches the result so the
// tunnel creation form can show a ping value before a tunnel exists. xray-core's observatory
// only runs inside a started xray instance and is therefore not usable on the creation form
// (issue #6).
//
// The probe is a plain net.DialTimeout to node.Address:node.Port. We deliberately do not
// perform a TLS/Reality handshake here:
//   - the goal is reachability, not protocol-level health;
//   - a TLS handshake on every probe would be wasteful and could trigger upstream rate limits;
//   - bare TCP is also what xray-core's observatory measures.
type NodeHealthMonitor struct {
	subs     domain.SubscriptionRepository
	interval time.Duration
	timeout  time.Duration
	parallel int

	dial func(ctx context.Context, network, addr string) (net.Conn, error) // injectable for tests

	mu    sync.RWMutex
	cache map[int64]NodeHealthSnapshot
}

// NewNodeHealthMonitor builds a monitor with the given polling interval (recommended 60s) and
// per-probe TCP timeout (recommended 3s).
func NewNodeHealthMonitor(subs domain.SubscriptionRepository, interval, timeout time.Duration, parallel int) *NodeHealthMonitor {
	if parallel <= 0 {
		parallel = 10
	}
	dialer := &net.Dialer{Timeout: timeout}
	return &NodeHealthMonitor{
		subs:     subs,
		interval: interval,
		timeout:  timeout,
		parallel: parallel,
		dial:     dialer.DialContext,
		cache:    make(map[int64]NodeHealthSnapshot),
	}
}

// Get returns the latest snapshot for a node, or (zero, false) if the node has never been
// probed yet.
func (m *NodeHealthMonitor) Get(nodeID int64) (NodeHealthSnapshot, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.cache[nodeID]
	return s, ok
}

// Snapshot returns a copy of the full cache for bulk API exposure.
func (m *NodeHealthMonitor) Snapshot() map[int64]NodeHealthSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[int64]NodeHealthSnapshot, len(m.cache))
	maps.Copy(out, m.cache)
	return out
}

// Run blocks until ctx is canceled, probing all nodes every interval. The first cycle starts
// immediately so the UI does not show "—" for the whole first interval after startup.
func (m *NodeHealthMonitor) Run(ctx context.Context) {
	m.probeAll(ctx)
	t := time.NewTicker(m.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.probeAll(ctx)
		}
	}
}

func (m *NodeHealthMonitor) probeAll(ctx context.Context) {
	nodes, err := m.subs.ListAllNodes(ctx)
	if err != nil {
		ctxlog.From(ctx).Warn().Err(err).Msg("node_health: ListAllNodes failed, keeping previous cache")
		return
	}
	if len(nodes) == 0 {
		return
	}

	sem := make(chan struct{}, m.parallel)
	var wg sync.WaitGroup
	for _, n := range nodes {
		if n == nil || n.Address == "" || n.Port <= 0 {
			continue
		}
		node := n
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			snap := m.probeOne(ctx, node.Address, node.Port)
			m.mu.Lock()
			m.cache[node.ID] = snap
			m.mu.Unlock()
		}()
	}
	wg.Wait()

	// Garbage-collect entries for nodes that no longer exist so deleted-node cache
	// rows do not accumulate indefinitely.
	alive := make(map[int64]struct{}, len(nodes))
	for _, n := range nodes {
		if n != nil {
			alive[n.ID] = struct{}{}
		}
	}
	m.mu.Lock()
	for id := range m.cache {
		if _, ok := alive[id]; !ok {
			delete(m.cache, id)
		}
	}
	m.mu.Unlock()
}

func (m *NodeHealthMonitor) probeOne(ctx context.Context, host string, port int) NodeHealthSnapshot {
	dialCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	start := time.Now()
	conn, err := m.dial(dialCtx, "tcp", addr)
	now := time.Now().UTC()
	if err != nil {
		return NodeHealthSnapshot{Alive: false, DelayMs: -1, SampledAt: now}
	}
	delay := time.Since(start).Milliseconds()
	_ = conn.Close()
	return NodeHealthSnapshot{Alive: true, DelayMs: delay, SampledAt: now}
}

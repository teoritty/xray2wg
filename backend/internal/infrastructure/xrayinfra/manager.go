package xrayinfra

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"xray2wg/backend/internal/domain"

	"github.com/rs/zerolog/log"
	observatory "github.com/xtls/xray-core/app/observatory"
	core "github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/features/extension"
	"github.com/xtls/xray-core/infra/conf/serial"

	// Register inbound/outbound handlers (proxyman, dokodemo, vless, …) when embedding xray as a library.
	_ "github.com/xtls/xray-core/main/distro/all"
)

// NodeHealthEntry reports the health of one VLESS outbound as seen by the xray observatory.
type NodeHealthEntry struct {
	Tag     string // e.g. "vless-out-1"
	Alive   bool
	DelayMs int64 // -1 = unknown (no probe data yet)
}

type Manager struct {
	mu        sync.RWMutex
	instances map[int]*core.Instance
}

func NewManager() *Manager {
	return &Manager{instances: make(map[int]*core.Instance)}
}

func (m *Manager) Start(tunnelID int, xrayPort int, fwmark int, localGatewayIP string, nodes []*domain.VlessNode, strategy domain.BalancingStrategy) error {
	if len(nodes) == 0 {
		return fmt.Errorf("xray Start: no nodes provided for tunnel %d", tunnelID)
	}
	log.Info().
		Int("tunnel_id", tunnelID).
		Int("xray_listen_port", xrayPort).
		Int("fwmark", fwmark).
		Str("local_gateway", localGatewayIP).
		Int("node_count", len(nodes)).
		Str("strategy", string(strategy)).
		Str("vless_remote", nodes[0].Address).
		Msg("tunnel_trace xray Start: build config")

	raw, err := BuildXrayConfig(xrayPort, fwmark, localGatewayIP, nodes, strategy)
	if err != nil {
		log.Error().Int("tunnel_id", tunnelID).Err(err).Msg("tunnel_trace xray Start: BuildXrayConfig failed")
		return err
	}
	log.Info().Int("tunnel_id", tunnelID).Int("config_json_bytes", len(raw)).Msg("tunnel_trace xray Start: JSON ready")

	cfg, err := serial.LoadJSONConfig(bytes.NewReader(raw))
	if err != nil {
		log.Error().Int("tunnel_id", tunnelID).Err(err).Msg("tunnel_trace xray Start: LoadJSONConfig failed")
		return err
	}
	log.Info().Int("tunnel_id", tunnelID).Msg("tunnel_trace xray Start: LoadJSONConfig ok")

	inst, err := core.New(cfg)
	if err != nil {
		log.Error().Int("tunnel_id", tunnelID).Err(err).Msg("tunnel_trace xray Start: core.New failed")
		return err
	}
	log.Info().Int("tunnel_id", tunnelID).Msg("tunnel_trace xray Start: core.New ok")

	if err := inst.Start(); err != nil {
		_ = inst.Close()
		log.Error().Int("tunnel_id", tunnelID).Err(err).Msg("tunnel_trace xray Start: inst.Start failed")
		return err
	}
	log.Info().Int("tunnel_id", tunnelID).Msg("tunnel_trace xray Start: inst.Start ok (inbound listening)")

	m.mu.Lock()
	defer m.mu.Unlock()
	old := m.instances[tunnelID]
	if old != nil {
		_ = old.Close()
	}
	m.instances[tunnelID] = inst
	return nil
}

// NodeHealth returns observatory health data for the tunnel's VLESS outbounds.
// Returns nil if the tunnel is not running or observatory is not enabled (e.g. round-robin single node).
func (m *Manager) NodeHealth(tunnelID int) []NodeHealthEntry {
	m.mu.RLock()
	inst := m.instances[tunnelID]
	m.mu.RUnlock()
	if inst == nil {
		return nil
	}
	obs, ok := inst.GetFeature(extension.ObservatoryType()).(extension.Observatory)
	if !ok || obs == nil {
		return nil
	}
	msg, err := obs.GetObservation(context.Background())
	if err != nil || msg == nil {
		return nil
	}
	result, ok := msg.(*observatory.ObservationResult)
	if !ok || result == nil {
		return nil
	}
	out := make([]NodeHealthEntry, 0, len(result.Status))
	for _, s := range result.Status {
		delay := s.Delay
		if !s.Alive {
			delay = -1
		}
		out = append(out, NodeHealthEntry{
			Tag:     s.OutboundTag,
			Alive:   s.Alive,
			DelayMs: delay,
		})
	}
	return out
}

func (m *Manager) Stop(tunnelID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	inst := m.instances[tunnelID]
	if inst == nil {
		log.Info().Int("tunnel_id", tunnelID).Msg("tunnel_trace xray Stop: no instance")
		return nil
	}
	delete(m.instances, tunnelID)
	log.Info().Int("tunnel_id", tunnelID).Msg("tunnel_trace xray Stop: closing instance")
	return inst.Close()
}

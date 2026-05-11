package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"xray2wg/backend/internal/ctxlog"
	"xray2wg/backend/internal/domain"
	cryptoutil "xray2wg/backend/internal/infrastructure/crypto"
	"xray2wg/backend/internal/infrastructure/netconf"
	"xray2wg/backend/internal/telemetry"
	"xray2wg/backend/internal/vless"
	"xray2wg/backend/internal/wgkey"
)

type TunnelService struct {
	Tunnels   domain.TunnelRepository
	Peers     domain.PeerRepository
	Subs      domain.SubscriptionRepository
	Xray      XrayProcess
	WG        WgTunnel
	MasterKey []byte
	Events    *EventLog
	Stats     *StatsCollector

	running *RunSet
}

func NewTunnelService(tr domain.TunnelRepository, pr domain.PeerRepository, subs domain.SubscriptionRepository,
	xr XrayProcess, wg WgTunnel, key []byte, ev *EventLog, stats *StatsCollector,
) *TunnelService {
	return &TunnelService{
		Tunnels: tr, Peers: pr, Subs: subs, Xray: xr, WG: wg, MasterKey: key,
		Events: ev, Stats: stats, running: NewRunSet(),
	}
}

func (t *TunnelService) refreshTunnelGauge() {
	if telemetry.ActiveTunnels == nil {
		return
	}
	telemetry.ActiveTunnels.Set(float64(t.running.Len()))
}

// logTeardownError logs a non-fatal persistence failure during tunnel lifecycle (plan naming: teardown errors must not be silently ignored).
func logTeardownError(ctx context.Context, tunnelID int64, op string, err error) {
	if err == nil {
		return
	}
	ctxlog.From(ctx).Warn().Int64("tunnel_id", tunnelID).Str("op", op).Err(err).Msg("tunnel DB update failed")
}

func (t *TunnelService) IsRunning(id int64) bool {
	return t.running.Contains(id)
}

func (t *TunnelService) computeRuntime(id int64) (tunName string, xrayPort, fwmark, routeTable int) {
	tunName = fmt.Sprintf("tun-wg%d", id)
	xrayPort = 13000 + int(id)
	fwmark = int(0x1000 + id)
	routeTable = 100 + int(id)
	return
}

// ReloadNodes hot-reloads the xray config when a tunnel's node list changes.
// WireGuard device and iptables rules are unaffected; only xray restarts.
func (t *TunnelService) ReloadNodes(ctx context.Context, id int64) error {
	if !t.IsRunning(id) {
		return nil
	}
	iface, _, err := t.Tunnels.GetByID(ctx, id)
	if err != nil {
		return err
	}
	nodes, err := t.Tunnels.ListNodes(ctx, id)
	if err != nil || len(nodes) == 0 {
		return fmt.Errorf("ReloadNodes: no nodes assigned to tunnel %d", id)
	}
	parsed := make([]*domain.VlessNode, 0, len(nodes))
	for _, n := range nodes {
		p, pErr := vless.ParseURI(n.RawURI)
		if pErr != nil || p == nil {
			p = n
		}
		parsed = append(parsed, p)
	}
	_, xrayPort, fwmark, _ := t.computeRuntime(id)
	localGW := ""
	if gw, gwErr := netconf.WgGatewayHost(iface.WgAddress); gwErr == nil {
		localGW = gw
	}
	strategy := iface.BalancingStrategy
	if strategy == "" {
		strategy = domain.BalancingRoundRobin
	}
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Int("node_count", len(parsed)).Str("strategy", string(strategy)).Msg("tunnel_trace ReloadNodes: restarting xray")
	return t.Xray.Start(int(id), xrayPort, fwmark, localGW, parsed, strategy)
}

func (t *TunnelService) Create(ctx context.Context, name string, listen int, wgAddr string, dns string, mtu int,
	subID *int64, nodeID *int64, nodeIDs []int64, strategy domain.BalancingStrategy,
) (*domain.WgInterface, error) {
	priv, pub, err := wgkey.GenerateKeypair()
	if err != nil {
		return nil, err
	}
	enc, err := cryptoutil.EncryptGCM(t.MasterKey, []byte(priv))
	if err != nil {
		return nil, err
	}
	if mtu <= 0 {
		mtu = 1420
	}
	if strategy == "" {
		strategy = domain.BalancingRoundRobin
	}
	userDNS := strings.TrimSpace(dns)
	iface := &domain.WgInterface{
		Name:              name,
		ListenPort:        listen,
		PublicKey:         pub,
		WgAddress:         wgAddr, // filled after ID if empty
		DNS:               userDNS,
		MTU:               mtu,
		SubscriptionID:    subID,
		ActiveNodeID:      nodeID,
		BalancingStrategy: strategy,
		Status:            domain.WgStatusStopped,
	}
	if err := t.Tunnels.Create(ctx, iface, enc); err != nil {
		return nil, err
	}
	if strings.TrimSpace(wgAddr) == "" {
		iface.WgAddress = fmt.Sprintf("10.100.%d.1/24", iface.ID)
	}
	if strings.TrimSpace(iface.DNS) == "" {
		iface.DNS = fmt.Sprintf("10.100.%d.1", iface.ID)
	}
	tunName, xrayPort, fwmark, _ := t.computeRuntime(iface.ID)
	iface.TunName = tunName
	iface.XrayPort = xrayPort
	iface.FWMark = fwmark
	logTeardownError(ctx, iface.ID, "Create.Update", t.Tunnels.Update(ctx, iface))
	logTeardownError(ctx, iface.ID, "Create.UpdateRuntimeFields", t.Tunnels.UpdateRuntimeFields(ctx, iface.ID, tunName, xrayPort, fwmark))

	// Populate junction table. nodeIDs takes priority; fall back to single nodeID.
	effectiveNodeIDs := nodeIDs
	if len(effectiveNodeIDs) == 0 && nodeID != nil {
		effectiveNodeIDs = []int64{*nodeID}
	}
	if len(effectiveNodeIDs) > 0 {
		if err := t.Tunnels.SetNodes(ctx, iface.ID, effectiveNodeIDs); err != nil {
			ctxlog.From(ctx).Warn().Int64("tunnel_id", iface.ID).Err(err).Msg("Create: SetNodes failed")
		}
	}

	t.Events.Add("info", fmt.Sprintf("tunnel created: %s (%d)", name, iface.ID))
	return iface, nil
}

func (t *TunnelService) Start(ctx context.Context, id int64, peerSvc *PeerService) error {
	if t.running.Contains(id) {
		ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace Start: already running, skip")
		return nil
	}
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace Start: begin")

	iface, privEnc, err := t.Tunnels.GetByID(ctx, id)
	if err != nil {
		ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: GetByID failed")
		return err
	}
	// Resolve nodes: prefer the junction table; fall back to legacy ActiveNodeID.
	activeNodes, err := t.Tunnels.ListNodes(ctx, id)
	if err != nil {
		ctxlog.From(ctx).Warn().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: ListNodes failed, falling back to ActiveNodeID")
		activeNodes = nil
	}
	if len(activeNodes) == 0 {
		// Legacy single-node path.
		if iface.ActiveNodeID == nil && iface.SubscriptionID != nil {
			subNodes, listErr := t.Subs.ListNodes(ctx, *iface.SubscriptionID)
			if listErr == nil && len(subNodes) == 1 {
				only := subNodes[0].ID
				iface.ActiveNodeID = &only
				ctxlog.From(ctx).Warn().Int64("tunnel_id", id).Int64("active_node_id", only).
					Msg("tunnel_trace Start: restored active_node_id (single node for subscription)")
				if err := t.Tunnels.Update(ctx, iface); err != nil {
					ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: persist restored active_node_id failed")
				}
			}
		}
		if iface.ActiveNodeID == nil {
			ctxlog.From(ctx).Error().Int64("tunnel_id", id).Msg("tunnel_trace Start: active node missing")
			return fmt.Errorf("active node required")
		}
		legacyNode, err := t.Subs.GetNode(ctx, *iface.ActiveNodeID)
		if err != nil {
			ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: GetNode failed")
			return err
		}
		parsed, err := vless.ParseURI(legacyNode.RawURI)
		if err != nil || parsed == nil {
			parsed = legacyNode
		}
		activeNodes = []*domain.VlessNode{parsed}
	} else {
		// Parse raw URIs for all junction nodes.
		parsed := make([]*domain.VlessNode, 0, len(activeNodes))
		for _, n := range activeNodes {
			p, err := vless.ParseURI(n.RawURI)
			if err != nil || p == nil {
				p = n
			}
			parsed = append(parsed, p)
		}
		activeNodes = parsed
	}
	// Use first node's flow for iptables TPROXY hint (all nodes from same subscription share the same protocol).
	representativeFlow := ""
	if len(activeNodes) > 0 {
		representativeFlow = activeNodes[0].Flow
	}
	strategy := iface.BalancingStrategy
	if strategy == "" {
		strategy = domain.BalancingRoundRobin
	}

	privStrB, err := cryptoutil.DecryptGCM(t.MasterKey, privEnc)
	if err != nil {
		ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: decrypt server key failed")
		return err
	}
	privStr := strings.TrimSpace(string(privStrB))

	tunName, xrayPort, fwmark, rtbl := t.computeRuntime(id)
	peers, err := t.Peers.ListByInterface(ctx, id)
	if err != nil {
		ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: ListByInterface failed")
		return err
	}
	psk := peerSvc.PSKCallback(ctx, id)

	masqSubnet, err := netconf.ClientSubnetCIDR(iface.WgAddress)
	if err != nil {
		ctxlog.From(ctx).Error().Err(err).Str("wg_address", iface.WgAddress).Msg("tunnel_trace Start: WgAddress CIDR invalid for NAT")
		return fmt.Errorf("invalid wg address: %w", err)
	}

	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Str("tun", tunName).Int("wg_listen", iface.ListenPort).Int("mtu", iface.MTU).Int("peer_count", len(peers)).Msg("tunnel_trace Start: step wg_create")
	if err := t.WG.Create(int(id), iface.MTU, iface.ListenPort, privStr, peers, psk, tunName); err != nil {
		ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: WG.Create failed")
		logTeardownError(ctx, id, "Start.UpdateStatus", t.Tunnels.UpdateStatus(ctx, id, domain.WgStatusError, err.Error()))
		return err
	}
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Str("tun", tunName).Msg("tunnel_trace Start: step wg_create ok")

	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Str("tun", tunName).Str("cidr", iface.WgAddress).Msg("tunnel_trace Start: step assign_tun")
	if err := netconf.AssignTUN(tunName, iface.WgAddress); err != nil {
		ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: AssignTUN failed")
		t.WG.Destroy(int(id))
		logTeardownError(ctx, id, "Start.UpdateStatus", t.Tunnels.UpdateStatus(ctx, id, domain.WgStatusError, err.Error()))
		return err
	}
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace Start: step assign_tun ok")

	netconf.PrepareTunForTransparentProxy(tunName)
	netconf.LogIPv4ForwardingSnapshot(tunName)

	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Int("fwmark", fwmark).Int("route_table", rtbl).Msg("tunnel_trace Start: step policy_route")
	if err := netconf.SetupReturnRoute(fwmark, rtbl); err != nil {
		ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: SetupReturnRoute failed")
		t.WG.Destroy(int(id))
		logTeardownError(ctx, id, "Start.UpdateStatus", t.Tunnels.UpdateStatus(ctx, id, domain.WgStatusError, err.Error()))
		return err
	}
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace Start: step policy_route ok")

	localGW := ""
	if gw, err := netconf.WgGatewayHost(iface.WgAddress); err != nil {
		ctxlog.From(ctx).Warn().Err(err).Str("wg_address", iface.WgAddress).Msg("tunnel_trace Start: WgGatewayHost failed, DNS inbound skipped")
	} else {
		localGW = gw
	}
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Int("xray_port", xrayPort).Str("vless", activeNodes[0].Address).Int("node_count", len(activeNodes)).Str("strategy", string(strategy)).Str("local_gateway", localGW).Msg("tunnel_trace Start: step xray_embedded")
	if err := t.Xray.Start(int(id), xrayPort, fwmark, localGW, activeNodes, strategy); err != nil {
		ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: Xray.Start failed")
		_ = netconf.TeardownReturnRoute(fwmark, rtbl)
		t.WG.Destroy(int(id))
		logTeardownError(ctx, id, "Start.UpdateStatus", t.Tunnels.UpdateStatus(ctx, id, domain.WgStatusError, err.Error()))
		return err
	}
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace Start: step xray_embedded ok")

	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Str("tun", tunName).Int("xray_port", xrayPort).Int("fwmark", fwmark).Msg("tunnel_trace Start: step iptables_tproxy")
	if err := netconf.SetupTProxy(int(id), tunName, xrayPort, fwmark, localGW, representativeFlow); err != nil {
		ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: SetupTProxy failed")
		_ = t.Xray.Stop(int(id))
		_ = netconf.TeardownReturnRoute(fwmark, rtbl)
		t.WG.Destroy(int(id))
		logTeardownError(ctx, id, "Start.UpdateStatus", t.Tunnels.UpdateStatus(ctx, id, domain.WgStatusError, err.Error()))
		return err
	}
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace Start: step iptables_tproxy ok")

	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Str("tun", tunName).Msg("tunnel_trace Start: step iptables_forward")
	if err := netconf.SetupForwardRules(tunName); err != nil {
		ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: SetupForwardRules failed")
		_ = netconf.TeardownTProxy(int(id), tunName, fwmark)
		_ = t.Xray.Stop(int(id))
		_ = netconf.TeardownReturnRoute(fwmark, rtbl)
		t.WG.Destroy(int(id))
		logTeardownError(ctx, id, "Start.UpdateStatus", t.Tunnels.UpdateStatus(ctx, id, domain.WgStatusError, err.Error()))
		return err
	}
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace Start: step iptables_forward ok")

	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Str("subnet", masqSubnet).Msg("tunnel_trace Start: step iptables_nat_masq")
	if err := netconf.SetupNATMasquerade(tunName, masqSubnet); err != nil {
		ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace Start: SetupNATMasquerade failed")
		_ = netconf.TeardownForwardRules(tunName)
		_ = netconf.TeardownTProxy(int(id), tunName, fwmark)
		_ = t.Xray.Stop(int(id))
		_ = netconf.TeardownReturnRoute(fwmark, rtbl)
		t.WG.Destroy(int(id))
		logTeardownError(ctx, id, "Start.UpdateStatus", t.Tunnels.UpdateStatus(ctx, id, domain.WgStatusError, err.Error()))
		return err
	}
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace Start: step iptables_nat_masq ok")

	now := time.Now().UTC()
	iface.Status = domain.WgStatusRunning
	iface.ErrorMessage = ""
	iface.UptimeStarted = &now
	iface.TunName = tunName
	iface.XrayPort = xrayPort
	iface.FWMark = fwmark
	if err := t.Tunnels.Update(ctx, iface); err != nil {
		t.stopInternal(ctx, id, tunName, fwmark, rtbl, masqSubnet)
		return err
	}
	t.running.Mark(id)
	t.refreshTunnelGauge()
	if t.Stats != nil {
		t.Stats.Track(id)
	}
	t.Events.Add("info", fmt.Sprintf("tunnel started: %s (%d)", iface.Name, id))
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Str("name", iface.Name).Str("tun", tunName).Msg("tunnel_trace Start: complete — tunnel running")
	return nil
}

func (t *TunnelService) stopInternal(ctx context.Context, id int64, tunName string, fwmark, rtbl int, masqSubnet string) {
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Str("tun", tunName).Msg("tunnel_trace Stop: teardown forward rules")
	_ = netconf.TeardownForwardRules(tunName)
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Str("subnet", masqSubnet).Msg("tunnel_trace Stop: teardown nat masq")
	_ = netconf.TeardownNATMasquerade(tunName, masqSubnet)
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace Stop: teardown tproxy")
	_ = netconf.TeardownTProxy(int(id), tunName, fwmark)
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace Stop: teardown policy route")
	_ = netconf.TeardownReturnRoute(fwmark, rtbl)
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace Stop: destroy wireguard device")
	t.WG.Destroy(int(id))
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace Stop: stop xray")
	_ = t.Xray.Stop(int(id))
	t.running.Clear(id)
	t.refreshTunnelGauge()
	if t.Stats != nil {
		t.Stats.Untrack(id)
	}
}

func (t *TunnelService) teardownRuntime(ctx context.Context, id int64, iface *domain.WgInterface) {
	_, _, fwmark, rtbl := t.computeRuntime(id)
	tunName := iface.TunName
	if tunName == "" {
		tunName = fmt.Sprintf("tun-wg%d", id)
	}
	masqSubnet, err := netconf.ClientSubnetCIDR(iface.WgAddress)
	if err != nil {
		ctxlog.From(ctx).Warn().Err(err).Str("wg_address", iface.WgAddress).Msg("tunnel_trace Stop: NAT subnet parse failed, NAT teardown skipped")
		masqSubnet = ""
	}
	t.stopInternal(ctx, id, tunName, fwmark, rtbl, masqSubnet)
}

func (t *TunnelService) Stop(ctx context.Context, id int64) error {
	iface, _, err := t.Tunnels.GetByID(ctx, id)
	if err != nil {
		return err
	}
	t.teardownRuntime(ctx, id, iface)

	iface.Status = domain.WgStatusStopped
	iface.ErrorMessage = ""
	iface.UptimeStarted = nil
	logTeardownError(ctx, id, "Stop.Update", t.Tunnels.Update(ctx, iface))
	logTeardownError(ctx, id, "Stop.UpdateStatus", t.Tunnels.UpdateStatus(ctx, id, domain.WgStatusStopped, ""))
	t.Events.Add("info", fmt.Sprintf("tunnel stopped (%d)", id))
	return nil
}

func (t *TunnelService) RestoreRunning(ctx context.Context, peerSvc *PeerService) {
	ids, err := t.Tunnels.ListRunningIDs(ctx)
	if err != nil {
		return
	}
	for _, id := range ids {
		ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace RestoreRunning: starting tunnel from DB")
		if err := t.Start(ctx, id, peerSvc); err != nil {
			ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace RestoreRunning: Start failed")
			t.Events.Add("error", fmt.Sprintf("restore tunnel %d: %v", id, err))
		}
	}
}

func (t *TunnelService) ShutdownAll(ctx context.Context) {
	for _, id := range t.running.Drain() {
		iface, _, err := t.Tunnels.GetByID(ctx, id)
		if err != nil {
			ctxlog.From(ctx).Warn().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace ShutdownAll: GetByID failed")
			continue
		}
		t.teardownRuntime(ctx, id, iface)
	}
	t.refreshTunnelGauge()
}

func (t *TunnelService) ReloadPeers(ctx context.Context, id int64, peerSvc *PeerService) error {
	if !t.IsRunning(id) {
		return nil
	}
	ctxlog.From(ctx).Info().Int64("tunnel_id", id).Msg("tunnel_trace ReloadPeers: applying new peer set to wireguard")
	iface, privEnc, err := t.Tunnels.GetByID(ctx, id)
	if err != nil {
		return err
	}
	skB, err := cryptoutil.DecryptGCM(t.MasterKey, privEnc)
	if err != nil {
		return err
	}
	peers, err := t.Peers.ListByInterface(ctx, id)
	if err != nil {
		return err
	}
	err = t.WG.ReloadPeers(int(id), iface.MTU, iface.ListenPort, strings.TrimSpace(string(skB)),
		peers, peerSvc.PSKCallback(ctx, id))
	if err != nil {
		ctxlog.From(ctx).Error().Int64("tunnel_id", id).Err(err).Msg("tunnel_trace ReloadPeers: failed")
	} else {
		ctxlog.From(ctx).Info().Int64("tunnel_id", id).Int("peer_count", len(peers)).Msg("tunnel_trace ReloadPeers: ok")
	}
	return err
}

package app

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"xray2wg/backend/internal/domain"
	cryptoutil "xray2wg/backend/internal/infrastructure/crypto"
	"xray2wg/backend/internal/infrastructure/netconf"
	"xray2wg/backend/internal/service"
	"xray2wg/backend/internal/vless"
)

// TunnelsAPI implements tunnel-related HTTP use-cases (mount only binds Echo ↔ this type).
type TunnelsAPI struct {
	TunRepo     domain.TunnelRepository
	SubRepo     domain.SubscriptionRepository
	PeerRepo    domain.PeerRepository
	Set         domain.SettingRepository
	Tunnels     *service.TunnelService
	Peers       *service.PeerService
	Stats       domain.StatsRepository
	EventLog    *service.EventLog
	MasterKey   []byte
	ManualSubID int64
	meta        *TunnelsApp
}

// NewTunnelsAPI constructs tunnel HTTP use-cases.
func NewTunnelsAPI(
	tunRepo domain.TunnelRepository,
	subRepo domain.SubscriptionRepository,
	peerRepo domain.PeerRepository,
	set domain.SettingRepository,
	tunnelSvc *service.TunnelService,
	peers *service.PeerService,
	stats domain.StatsRepository,
	el *service.EventLog,
	masterKey []byte,
	manualSubID int64,
) *TunnelsAPI {
	return &TunnelsAPI{
		TunRepo:     tunRepo,
		SubRepo:     subRepo,
		PeerRepo:    peerRepo,
		Set:         set,
		Tunnels:     tunnelSvc,
		Peers:       peers,
		Stats:       stats,
		EventLog:    el,
		MasterKey:   masterKey,
		ManualSubID: manualSubID,
		meta:        NewTunnelsApp(tunRepo, tunnelSvc),
	}
}

// ListTunnels returns all WireGuard interfaces.
func (a *TunnelsAPI) ListTunnels(ctx context.Context) ([]*domain.WgInterface, error) {
	return a.TunRepo.List(ctx)
}

// CreateTunnelInput is the JSON body for POST /tunnels.
type CreateTunnelInput struct {
	Name              string
	ListenPort        int
	WgAddress         string
	DNS               string
	MTU               int
	MSSClamp          int
	SubscriptionID    *int64
	ActiveNodeID      *int64
	NodeIDs           []int64
	BalancingStrategy string
	VlessURI          string
}

// CreateTunnel creates a tunnel, optionally importing a manual VLESS URI into the manual subscription.
func (a *TunnelsAPI) CreateTunnel(ctx context.Context, in CreateTunnelInput) (*domain.WgInterface, error) {
	subID := in.SubscriptionID
	nodeID := in.ActiveNodeID
	nodeIDs := in.NodeIDs
	if strings.TrimSpace(in.VlessURI) != "" {
		node, err := vless.ParseURI(strings.TrimSpace(in.VlessURI))
		if err != nil {
			return nil, err
		}
		node.SubscriptionID = a.ManualSubID
		if err := a.SubRepo.InsertNodes(ctx, []*domain.VlessNode{node}); err != nil {
			return nil, err
		}
		nodes, _ := a.SubRepo.ListNodes(ctx, a.ManualSubID)
		var nid int64
		for i := len(nodes) - 1; i >= 0; i-- {
			if nodes[i].RawURI == node.RawURI {
				nid = nodes[i].ID
				break
			}
		}
		tmp := nid
		nodeID = &tmp
		ms := a.ManualSubID
		subID = &ms
	}
	listenPort := in.ListenPort
	if listenPort == 0 {
		list, _ := a.TunRepo.List(ctx)
		listenPort = 51820 + len(list)
	}
	strategy := domain.BalancingStrategy(in.BalancingStrategy)
	return a.Tunnels.Create(ctx, in.Name, listenPort, in.WgAddress, in.DNS, in.MTU, in.MSSClamp, subID, nodeID, nodeIDs, strategy)
}

// GetTunnelNodes returns the ordered list of VLESS nodes for a tunnel with health data.
func (a *TunnelsAPI) GetTunnelNodes(ctx context.Context, tunnelID int64) (map[string]any, error) {
	iface, _, err := a.TunRepo.GetByID(ctx, tunnelID)
	if err != nil {
		return nil, err
	}
	nodes, err := a.TunRepo.ListNodes(ctx, tunnelID)
	if err != nil {
		return nil, err
	}

	// Fetch health from xray observatory if available.
	healthByTag := map[string]*domain.NodeHealthEntry{}
	if healthEntries := a.Tunnels.Xray.NodeHealth(int(tunnelID)); healthEntries != nil {
		for i := range healthEntries {
			e := healthEntries[i]
			healthByTag[e.Tag] = &e
		}
	}

	nodeList := make([]map[string]any, 0, len(nodes))
	for i, n := range nodes {
		tag := fmt.Sprintf("vless-out-%d", i+1)
		var healthObj any
		if h, ok := healthByTag[tag]; ok {
			healthObj = map[string]any{"alive": h.Alive, "delay_ms": h.DelayMs}
		}
		nodeList = append(nodeList, map[string]any{
			"id":           n.ID,
			"subscription_id": n.SubscriptionID,
			"display_name": n.DisplayName,
			"address":      n.Address,
			"port":         n.Port,
			"position":     i,
			"health":       healthObj,
		})
	}

	strategy := iface.BalancingStrategy
	if strategy == "" {
		strategy = domain.BalancingRoundRobin
	}
	return map[string]any{
		"strategy": string(strategy),
		"nodes":    nodeList,
	}, nil
}

// SetTunnelNodes assigns an ordered list of VLESS nodes to a tunnel and hot-reloads xray if running.
func (a *TunnelsAPI) SetTunnelNodes(ctx context.Context, tunnelID int64, nodeIDs []int64, strategy string) error {
	// Validate each node exists.
	for _, nid := range nodeIDs {
		if _, err := a.SubRepo.GetNode(ctx, nid); err != nil {
			return fmt.Errorf("node %d not found: %w", nid, err)
		}
	}

	if err := a.TunRepo.SetNodes(ctx, tunnelID, nodeIDs); err != nil {
		return err
	}

	// Persist strategy on the tunnel.
	if strategy != "" {
		iface, _, err := a.TunRepo.GetByID(ctx, tunnelID)
		if err != nil {
			return err
		}
		iface.BalancingStrategy = domain.BalancingStrategy(strategy)
		if err := a.TunRepo.Update(ctx, iface); err != nil {
			return err
		}
	}

	// Hot-reload xray if the tunnel is running.
	return a.Tunnels.ReloadNodes(ctx, tunnelID)
}

// StatsSummary builds the JSON payload for GET /stats/summary.
func (a *TunnelsAPI) StatsSummary(ctx context.Context) (map[string]any, error) {
	tr, pr, trx, tt, err := a.Stats.DBCounts(ctx)
	if err != nil {
		return nil, err
	}
	ev := a.EventLog.List()
	rt, _ := a.Stats.QueryLatestInterfaceRates(ctx)
	el := len(ev)
	if el > 20 {
		el = 20
	}
	rev := make([]service.EventEntry, el)
	for i := 0; i < el; i++ {
		rev[i] = ev[len(ev)-1-i]
	}
	return map[string]any{
		"active_tunnels": tr,
		"total_peers":    pr,
		"total_rx":       trx,
		"total_tx":       tt,
		"tunnel_rates":   rt,
		"events":         rev,
	}, nil
}

// GetTunnel returns one interface by id.
func (a *TunnelsAPI) GetTunnel(ctx context.Context, id int64) (*domain.WgInterface, error) {
	iface, _, err := a.TunRepo.GetByID(ctx, id)
	return iface, err
}

// UpdateTunnel merges PUT body, applies invariants, default DNS, and persists.
func (a *TunnelsAPI) UpdateTunnel(ctx context.Context, id int64, iface *domain.WgInterface) error {
	existing, _, err := a.TunRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if err := a.meta.PrepareTunnelPUT(ctx, id, existing, iface); err != nil {
		return err
	}
	if strings.TrimSpace(iface.DNS) == "" {
		if gw, err := netconf.WgGatewayHost(iface.WgAddress); err == nil {
			iface.DNS = gw
		}
	}
	return a.TunRepo.Update(ctx, iface)
}

// DeleteTunnel stops a running tunnel (best effort) and deletes the row.
func (a *TunnelsAPI) DeleteTunnel(ctx context.Context, id int64) error {
	_ = a.Tunnels.Stop(ctx, id)
	return a.TunRepo.Delete(ctx, id)
}

func (a *TunnelsAPI) StartTunnel(ctx context.Context, id int64) error {
	return a.Tunnels.Start(ctx, id, a.Peers)
}

func (a *TunnelsAPI) StopTunnel(ctx context.Context, id int64) error {
	return a.Tunnels.Stop(ctx, id)
}

// TunnelStats returns rate samples for the given window label.
func (a *TunnelsAPI) TunnelStats(ctx context.Context, id int64, window string) ([]domain.StatSnapshot, error) {
	dur := time.Hour
	switch window {
	case "6h":
		dur = 6 * time.Hour
	case "24h":
		dur = 24 * time.Hour
	}
	to := time.Now().UTC()
	from := to.Add(-dur)
	return a.Stats.QueryInterfaceWindow(ctx, id, from, to)
}

// ListTunnelPeers lists peers for a tunnel.
func (a *TunnelsAPI) ListTunnelPeers(ctx context.Context, tunnelID int64) ([]*domain.WgPeer, error) {
	return a.PeerRepo.ListByInterface(ctx, tunnelID)
}

// CreateTunnelPeerInput is JSON for POST /tunnels/:id/peers.
type CreateTunnelPeerInput struct {
	Name          string
	PublicKey     string
	ClientAddress string
}

func (a *TunnelsAPI) CreateTunnelPeer(ctx context.Context, tunnelID int64, in CreateTunnelPeerInput) (*domain.WgPeer, error) {
	return a.Peers.Create(ctx, tunnelID, in.Name, in.PublicKey, in.ClientAddress)
}

// PeerClientConfig returns rendered client config with endpoint host resolved.
func (a *TunnelsAPI) PeerClientConfig(ctx context.Context, tunnelID, peerID int64) (string, error) {
	txt, err := a.Peers.ClientConfig(ctx, tunnelID, peerID)
	if err != nil {
		return "", err
	}
	sh, _ := a.Set.Get(ctx, "server_host")
	endpointHost := netconf.ResolvePeerEndpointHost(sh, strings.TrimSpace(os.Getenv("PUBLIC_HOST")))
	return strings.ReplaceAll(txt, service.WgPeerConfigEndpointHostPlaceholder, endpointHost), nil
}

// PeerMikrotikScript returns MikroTik CLI for a peer.
func (a *TunnelsAPI) PeerMikrotikScript(ctx context.Context, tunnelID, peerID int64) (string, error) {
	iface, _, err := a.TunRepo.GetByID(ctx, tunnelID)
	if err != nil {
		return "", err
	}
	peer, privEnc, pskEnc, err := a.PeerRepo.GetByID(ctx, tunnelID, peerID)
	if err != nil {
		return "", err
	}
	sh, _ := a.Set.Get(ctx, "server_host")
	endpointHost := netconf.ResolvePeerEndpointHost(sh, strings.TrimSpace(os.Getenv("PUBLIC_HOST")))
	privPlain, err := cryptoutil.DecryptGCM(a.MasterKey, privEnc)
	if err != nil {
		return "", err
	}
	pskPlain, err := cryptoutil.DecryptGCM(a.MasterKey, pskEnc)
	if err != nil {
		return "", err
	}
	out := service.BuildMikrotikCommands(iface, peer, endpointHost, strings.TrimSpace(string(privPlain)), strings.TrimSpace(string(pskPlain)))
	return out, nil
}

func (a *TunnelsAPI) UpdateTunnelPeer(ctx context.Context, peer *domain.WgPeer) error {
	return a.PeerRepo.Update(ctx, peer, nil, nil)
}

func (a *TunnelsAPI) DeleteTunnelPeer(ctx context.Context, tunnelID, peerID int64) error {
	if err := a.PeerRepo.Delete(ctx, tunnelID, peerID); err != nil {
		return err
	}
	return a.Tunnels.ReloadPeers(ctx, tunnelID, a.Peers)
}

// ParseID64 parses a path/query id parameter.
func ParseID64(raw string) (int64, error) {
	return strconv.ParseInt(raw, 10, 64)
}

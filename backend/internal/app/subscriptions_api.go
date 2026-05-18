package app

import (
	"context"
	"encoding/json"

	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/service"
	"xray2wg/backend/internal/vless"
	"xray2wg/backend/internal/vless/security"
	"xray2wg/backend/internal/vless/transport"
)

// SubscriptionsAPI is the use-case surface for /api/v1/subscriptions*.
type SubscriptionsAPI struct {
	SubRepo    domain.SubscriptionRepository
	Subs       *service.SubscriptionService
	NodeHealth *service.NodeHealthMonitor
}

func NewSubscriptionsAPI(repo domain.SubscriptionRepository, subs *service.SubscriptionService, nodeHealth *service.NodeHealthMonitor) *SubscriptionsAPI {
	return &SubscriptionsAPI{SubRepo: repo, Subs: subs, NodeHealth: nodeHealth}
}

// NodeHealthInfo mirrors the frontend NodeHealthInfo shape and is embedded in node listings so
// the tunnel creation form can render a ping badge before any tunnel exists (issue #6).
type NodeHealthInfo struct {
	Alive   bool  `json:"alive"`
	DelayMs int64 `json:"delay_ms"`
}

// VlessNodeWithHealth augments domain.VlessNode with the latest reachability probe and
// pre-decoded transport / security specs. Embedding the underlying node keeps the legacy
// JSON shape (Network, Security, RawURI, …) untouched; Transport and SecurityCfg expose
// the typed payloads frontend forms render without parsing raw JSON themselves.
type VlessNodeWithHealth struct {
	*domain.VlessNode
	Health      *NodeHealthInfo `json:"Health"`
	Transport   any             `json:"Transport"`
	SecurityCfg any             `json:"SecurityCfg"`
}

// decodedSpecsForNode returns the transport and security specs for a node, decoded with
// the registered implementations so the API DTO can expose them as typed JSON. A decode
// failure (corrupt JSON, unknown transport) is swallowed and the corresponding payload is
// returned as nil — the frontend treats nil as "fall back to RawURI".
func decodedSpecsForNode(n *domain.VlessNode) (tSpec, sSpec any) {
	if tr, err := transport.Default.Resolve(n.Network); err == nil {
		if spec, err := tr.DecodeSpec(n.TransportConfig); err == nil {
			tSpec = spec
		}
	}
	if sec, err := security.Default.Resolve(n.Security); err == nil {
		if spec, err := sec.DecodeSpec(n.SecurityConfig); err == nil {
			sSpec = spec
		}
	}
	return
}

func (a *SubscriptionsAPI) List(ctx context.Context) ([]*domain.Subscription, error) {
	return a.SubRepo.List(ctx)
}

type SubscriptionCreateInput struct {
	Name            string
	URL             string
	RefreshInterval int64
}

func (a *SubscriptionsAPI) Add(ctx context.Context, in SubscriptionCreateInput) (*domain.Subscription, error) {
	return a.Subs.Add(ctx, in.Name, in.URL, in.RefreshInterval)
}

func (a *SubscriptionsAPI) AddManualVlessNode(ctx context.Context, vlessURI string) (*domain.VlessNode, error) {
	return a.Subs.AddManualVlessNode(ctx, vlessURI)
}

func (a *SubscriptionsAPI) UpdateManualVlessNode(ctx context.Context, nodeID int64, vlessURI string) (*domain.VlessNode, error) {
	return a.Subs.UpdateManualVlessNode(ctx, nodeID, vlessURI)
}

// ManualNodeInput is the JSON body shape for structured manual-node creation. Every
// transport- and security-specific field lives inside the opaque Transport / SecurityCfg
// payloads so the API never needs to grow when a new transport is registered.
type ManualNodeInput struct {
	DisplayName    string          `json:"display_name"`
	UUID           string          `json:"uuid"`
	Address        string          `json:"address"`
	Port           int             `json:"port"`
	Flow           string          `json:"flow"`
	Encryption     string          `json:"encryption"`
	PacketEncoding string          `json:"packet_encoding"`
	Network        string          `json:"network"`
	Transport      json.RawMessage `json:"transport"`
	Security       string          `json:"security"`
	SecurityCfg    json.RawMessage `json:"security_cfg"`
}

func (in ManualNodeInput) toBuildInput() vless.BuildInput {
	return vless.BuildInput{
		DisplayName:     in.DisplayName,
		UUID:            in.UUID,
		Address:         in.Address,
		Port:            in.Port,
		Flow:            in.Flow,
		Encryption:      in.Encryption,
		PacketEncoding:  in.PacketEncoding,
		Network:         in.Network,
		TransportConfig: in.Transport,
		Security:        in.Security,
		SecurityConfig:  in.SecurityCfg,
	}
}

func (a *SubscriptionsAPI) AddManualNodeStructured(ctx context.Context, in ManualNodeInput) (*domain.VlessNode, error) {
	return a.Subs.AddManualNodeStructured(ctx, in.toBuildInput())
}

func (a *SubscriptionsAPI) UpdateManualNodeStructured(ctx context.Context, nodeID int64, in ManualNodeInput) (*domain.VlessNode, error) {
	return a.Subs.UpdateManualNodeStructured(ctx, nodeID, in.toBuildInput())
}

func (a *SubscriptionsAPI) DeleteManualVlessNode(ctx context.Context, nodeID int64) error {
	return a.Subs.DeleteManualVlessNode(ctx, nodeID)
}

func (a *SubscriptionsAPI) Get(ctx context.Context, id int64) (*domain.Subscription, error) {
	return a.SubRepo.GetByID(ctx, id)
}

func (a *SubscriptionsAPI) Update(ctx context.Context, su *domain.Subscription) error {
	return a.SubRepo.Update(ctx, su)
}

func (a *SubscriptionsAPI) Delete(ctx context.Context, id int64) error {
	a.Subs.StopRefreshLoop(id)
	return a.SubRepo.Delete(ctx, id)
}

func (a *SubscriptionsAPI) Refresh(ctx context.Context, id int64) error {
	return a.Subs.FetchAndUpdate(ctx, id)
}

func (a *SubscriptionsAPI) ListNodes(ctx context.Context, subID int64) ([]VlessNodeWithHealth, error) {
	nodes, err := a.SubRepo.ListNodes(ctx, subID)
	if err != nil {
		return nil, err
	}
	return a.enrichWithHealth(nodes), nil
}

func (a *SubscriptionsAPI) enrichWithHealth(nodes []*domain.VlessNode) []VlessNodeWithHealth {
	out := make([]VlessNodeWithHealth, 0, len(nodes))
	for _, n := range nodes {
		tSpec, sSpec := decodedSpecsForNode(n)
		entry := VlessNodeWithHealth{VlessNode: n, Transport: tSpec, SecurityCfg: sSpec}
		if a.NodeHealth != nil {
			if snap, ok := a.NodeHealth.Get(n.ID); ok {
				entry.Health = &NodeHealthInfo{Alive: snap.Alive, DelayMs: snap.DelayMs}
			}
		}
		out = append(out, entry)
	}
	return out
}

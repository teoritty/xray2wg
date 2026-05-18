package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"xray2wg/backend/internal/ctxlog"
	"xray2wg/backend/internal/domain"
	"xray2wg/backend/internal/vless"
)

const ManualSubscriptionName = "__manual__"
const UA = "xray2wg/1.0"

type SubscriptionService struct {
	repo   domain.SubscriptionRepository
	client *http.Client
	log    *EventLog

	mu      sync.Mutex
	stopped map[int64]context.CancelFunc
}

func NewSubscriptionService(repo domain.SubscriptionRepository, elog *EventLog) *SubscriptionService {
	return &SubscriptionService{
		repo: repo,
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("redirects")
				}
				return nil
			},
		},
		log:     elog,
		stopped: make(map[int64]context.CancelFunc),
	}
}

func (s *SubscriptionService) EnsureManual(ctx context.Context) (int64, error) {
	list, err := s.repo.List(ctx)
	if err != nil {
		return 0, err
	}
	for _, x := range list {
		if x.Name == ManualSubscriptionName {
			return x.ID, nil
		}
	}
	sub := &domain.Subscription{
		Name:            ManualSubscriptionName,
		URL:             "",
		RefreshInterval: 86400,
	}
	if err := s.repo.Create(ctx, sub); err != nil {
		return 0, err
	}
	return sub.ID, nil
}

func (s *SubscriptionService) StartRefreshLoop(ctx context.Context, id int64) {
	s.mu.Lock()
	if c, ok := s.stopped[id]; ok {
		c()
		delete(s.stopped, id)
	}
	subCtx, cancel := context.WithCancel(ctx)
	s.stopped[id] = cancel
	s.mu.Unlock()

	go s.refreshLoop(subCtx, id)
}

func (s *SubscriptionService) StopRefreshLoop(id int64) {
	s.mu.Lock()
	if c, ok := s.stopped[id]; ok {
		c()
		delete(s.stopped, id)
	}
	s.mu.Unlock()
}

func (s *SubscriptionService) refreshLoop(ctx context.Context, id int64) {
	t := time.NewTimer(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if err := s.FetchAndUpdate(ctx, id); err != nil {
				ctxlog.From(ctx).Warn().Err(err).Int64("sub", id).Msg("subscription fetch")
			}
			sub, err := s.repo.GetByID(ctx, id)
			d := time.Hour
			if err == nil && sub.RefreshInterval > 0 {
				d = time.Duration(sub.RefreshInterval) * time.Second
			}
			t.Reset(d)
		}
	}
}

func (s *SubscriptionService) FetchAndUpdate(ctx context.Context, id int64) error {
	sub, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if sub.Name == ManualSubscriptionName {
		return nil
	}
	if strings.TrimSpace(sub.URL) == "" {
		return domain.ErrValidation
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sub.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UA)
	resp, err := s.client.Do(req)
	if err != nil {
		sub.ErrorMessage = err.Error()
		if sub.NodeCount == 0 {
			sub.Status = domain.SubStatusError
		}
		s.log.Add("warn", fmt.Sprintf("subscription %q fetch failed: %s", sub.Name, err.Error()))
		_ = s.repo.Update(ctx, sub)
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return err
	}
	lines, err := decodeSubscriptionBody(body)
	if err != nil {
		sub.ErrorMessage = err.Error()
		if sub.NodeCount == 0 {
			sub.Status = domain.SubStatusError
		}
		s.log.Add("warn", fmt.Sprintf("subscription %q decode failed: %s", sub.Name, err.Error()))
		_ = s.repo.Update(ctx, sub)
		return err
	}
	var nodes []*domain.VlessNode
	for _, line := range lines {
		n, err := vless.ParseURI(line)
		if err != nil {
			ctxlog.From(ctx).Debug().Str("line", line).Msg("skip vless line")
			continue
		}
		n.SubscriptionID = id
		nodes = append(nodes, n)
	}

	bindings, err := s.repo.SnapshotActiveNodesForSubscription(ctx, id)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteNodes(ctx, id); err != nil {
		return err
	}
	if len(nodes) > 0 {
		if err := s.repo.InsertNodes(ctx, nodes); err != nil {
			return err
		}
		if err := s.repo.RemapActiveNodesAfterRefresh(ctx, id, bindings, nodes); err != nil {
			return err
		}
	}
	now := time.Now().UTC()
	sub.LastFetchedAt = &now
	sub.NodeCount = int64(len(nodes))
	sub.Status = domain.SubStatusActive
	sub.ErrorMessage = ""
	return s.repo.Update(ctx, sub)
}

func decodeSubscriptionBody(body []byte) ([]string, error) {
	text := strings.TrimSpace(string(body))
	cands := []string{text}
	if dec, err := base64.RawStdEncoding.DecodeString(text); err == nil {
		cands = append(cands, string(dec))
	}
	if dec, err := base64.StdEncoding.DecodeString(text); err == nil {
		cands = append(cands, string(dec))
	}
	for _, c := range cands {
		if v := extractVless(splitLines(c)); len(v) > 0 {
			return v, nil
		}
	}
	return nil, fmt.Errorf("no vless lines")
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.Split(s, "\n")
}

func extractVless(lines []string) []string {
	var vless []string
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if strings.HasPrefix(strings.ToLower(ln), "vless://") {
			vless = append(vless, ln)
		}
	}
	return vless
}

// AddManualVlessNode parses a single vless:// URI and appends it as a node under the manual subscription (__manual__).
func (s *SubscriptionService) AddManualVlessNode(ctx context.Context, raw string) (*domain.VlessNode, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, domain.ErrValidation
	}
	manualID, err := s.EnsureManual(ctx)
	if err != nil {
		return nil, err
	}
	node, err := vless.ParseURI(raw)
	if err != nil {
		return nil, err
	}
	node.SubscriptionID = manualID
	existing, err := s.repo.ListNodes(ctx, manualID)
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(node.RawURI)
	for _, e := range existing {
		if strings.TrimSpace(e.RawURI) == key {
			return nil, domain.ErrConflict
		}
	}
	if err := s.repo.InsertNodes(ctx, []*domain.VlessNode{node}); err != nil {
		return nil, err
	}
	if err := s.syncManualSubscriptionMeta(ctx, manualID); err != nil {
		return nil, err
	}
	return node, nil
}

// AddManualNodeStructured is the structured counterpart of AddManualVlessNode: it accepts
// fully-typed transport and security configurations instead of a vless:// URI string. The
// canonical RawURI is rebuilt internally so downstream code (subscription refresh,
// duplicate detection) keeps a single source of truth.
func (s *SubscriptionService) AddManualNodeStructured(ctx context.Context, in vless.BuildInput) (*domain.VlessNode, error) {
	node, err := vless.Build(in)
	if err != nil {
		return nil, err
	}
	manualID, err := s.EnsureManual(ctx)
	if err != nil {
		return nil, err
	}
	node.SubscriptionID = manualID
	existing, err := s.repo.ListNodes(ctx, manualID)
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(node.RawURI)
	for _, e := range existing {
		if strings.TrimSpace(e.RawURI) == key {
			return nil, domain.ErrConflict
		}
	}
	if err := s.repo.InsertNodes(ctx, []*domain.VlessNode{node}); err != nil {
		return nil, err
	}
	if err := s.syncManualSubscriptionMeta(ctx, manualID); err != nil {
		return nil, err
	}
	return node, nil
}

// UpdateManualNodeStructured mirrors UpdateManualVlessNode for structured input.
func (s *SubscriptionService) UpdateManualNodeStructured(ctx context.Context, nodeID int64, in vless.BuildInput) (*domain.VlessNode, error) {
	manualID, err := s.EnsureManual(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if existing.SubscriptionID != manualID {
		return nil, domain.ErrNotFound
	}
	node, err := vless.Build(in)
	if err != nil {
		return nil, err
	}
	node.SubscriptionID = manualID
	node.ID = nodeID
	others, err := s.repo.ListNodes(ctx, manualID)
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(node.RawURI)
	for _, e := range others {
		if e.ID != nodeID && strings.TrimSpace(e.RawURI) == key {
			return nil, domain.ErrConflict
		}
	}
	if err := s.repo.UpdateNode(ctx, node); err != nil {
		return nil, err
	}
	if err := s.syncManualSubscriptionMeta(ctx, manualID); err != nil {
		return nil, err
	}
	return s.repo.GetNode(ctx, nodeID)
}

// UpdateManualVlessNode replaces fields for an existing node under __manual__ from a new vless:// URI.
func (s *SubscriptionService) UpdateManualVlessNode(ctx context.Context, nodeID int64, raw string) (*domain.VlessNode, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, domain.ErrValidation
	}
	manualID, err := s.EnsureManual(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := s.repo.GetNode(ctx, nodeID)
	if err != nil {
		return nil, err
	}
	if existing.SubscriptionID != manualID {
		return nil, domain.ErrNotFound
	}
	parsed, err := vless.ParseURI(raw)
	if err != nil {
		return nil, err
	}
	parsed.SubscriptionID = manualID
	parsed.ID = nodeID

	others, err := s.repo.ListNodes(ctx, manualID)
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(parsed.RawURI)
	for _, e := range others {
		if e.ID != nodeID && strings.TrimSpace(e.RawURI) == key {
			return nil, domain.ErrConflict
		}
	}
	if err := s.repo.UpdateNode(ctx, parsed); err != nil {
		return nil, err
	}
	if err := s.syncManualSubscriptionMeta(ctx, manualID); err != nil {
		return nil, err
	}
	return s.repo.GetNode(ctx, nodeID)
}

// DeleteManualVlessNode removes a manual node if no tunnel references it as active_node_id.
func (s *SubscriptionService) DeleteManualVlessNode(ctx context.Context, nodeID int64) error {
	manualID, err := s.EnsureManual(ctx)
	if err != nil {
		return err
	}
	existing, err := s.repo.GetNode(ctx, nodeID)
	if err != nil {
		return err
	}
	if existing.SubscriptionID != manualID {
		return domain.ErrNotFound
	}
	refs, err := s.repo.FindTunnelIDsUsingNode(ctx, nodeID)
	if err != nil {
		return err
	}
	if len(refs) > 0 {
		return domain.ErrNodeInUse
	}
	if err := s.repo.DeleteNode(ctx, nodeID); err != nil {
		return err
	}
	return s.syncManualSubscriptionMeta(ctx, manualID)
}

func (s *SubscriptionService) syncManualSubscriptionMeta(ctx context.Context, manualID int64) error {
	nodes, err := s.repo.ListNodes(ctx, manualID)
	if err != nil {
		return err
	}
	sub, err := s.repo.GetByID(ctx, manualID)
	if err != nil {
		return err
	}
	sub.NodeCount = int64(len(nodes))
	sub.Status = domain.SubStatusActive
	sub.ErrorMessage = ""
	return s.repo.Update(ctx, sub)
}

func (s *SubscriptionService) Add(ctx context.Context, name, url string, refreshSec int64) (*domain.Subscription, error) {
	if refreshSec <= 0 {
		refreshSec = 3600
	}
	sub := &domain.Subscription{Name: name, URL: url, RefreshInterval: refreshSec}
	if err := s.repo.Create(ctx, sub); err != nil {
		return nil, err
	}
	if err := s.FetchAndUpdate(ctx, sub.ID); err != nil {
		s.log.Add("warning", fmt.Sprintf("subscription %s initial fetch: %v", name, err))
	}
	s.StartRefreshLoop(context.Background(), sub.ID)
	return sub, nil
}

type exportPayload struct {
	Subscriptions []struct {
		Name              string `json:"name"`
		URL               string `json:"url"`
		RefreshInterval   int64  `json:"refresh_interval"`
	} `json:"subscriptions"`
}

func (s *SubscriptionService) ExportMinimal(ctx context.Context) ([]byte, error) {
	subs, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	var p exportPayload
	for _, su := range subs {
		if su.Name == ManualSubscriptionName {
			continue
		}
		p.Subscriptions = append(p.Subscriptions, struct {
			Name            string `json:"name"`
			URL             string `json:"url"`
			RefreshInterval int64  `json:"refresh_interval"`
		}{Name: su.Name, URL: su.URL, RefreshInterval: su.RefreshInterval})
	}
	return json.MarshalIndent(p, "", "  ")
}

func (s *SubscriptionService) ImportMinimal(ctx context.Context, b []byte) error {
	var p exportPayload
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	for _, x := range p.Subscriptions {
		if _, err := s.Add(ctx, x.Name, x.URL, x.RefreshInterval); err != nil {
			return err
		}
	}
	return nil
}

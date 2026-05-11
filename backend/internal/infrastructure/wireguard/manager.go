package wireguardinfra

import (
	"encoding/hex"
	"fmt"
	"net"
	"strings"
	"sync"

	"xray2wg/backend/internal/domain"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"github.com/rs/zerolog/log"
)

// managedDevice keeps the userspace WireGuard device together with its UAPI
// listener. The UAPI socket is what wgctrl (and `wg show`) talk to in order to
// read live peer statistics for a userspace device — without it, the stats
// collector can never read rx/tx counters, and the Statistics / Tunnels pages
// stay frozen at zero.
type managedDevice struct {
	dev      *device.Device
	uapi     net.Listener
	uapiDone chan struct{}
}

type Manager struct {
	mu      sync.RWMutex
	devices map[int]*managedDevice
}

func NewManager() *Manager {
	return &Manager{devices: make(map[int]*managedDevice)}
}

func buildIPC(listenPort int, privB64 string, peers []*domain.WgPeer, pskForPub func(pub string) []byte) (string, error) {
	sk, err := wgtypes.ParseKey(strings.TrimSpace(privB64))
	if err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "private_key=%s\n", hex.EncodeToString(sk[:]))
	fmt.Fprintf(&b, "listen_port=%d\n", listenPort)
	// Align with WireGuard configuration protocol — see wireguard-go device/uapi.go
	fmt.Fprintf(&b, "replace_peers=true\n")
	for _, p := range peers {
		pub, err := wgtypes.ParseKey(strings.TrimSpace(p.PublicKey))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "public_key=%s\n", hex.EncodeToString(pub[:]))
		fmt.Fprintf(&b, "protocol_version=1\n")
		if pskForPub != nil {
			if pk := pskForPub(strings.TrimSpace(p.PublicKey)); len(pk) == 32 {
				fmt.Fprintf(&b, "preshared_key=%s\n", hex.EncodeToString(pk))
			}
		}
		fmt.Fprintf(&b, "replace_allowed_ips=true\n")
		for _, cidr := range splitAllowed(p.AllowedIPs) {
			fmt.Fprintf(&b, "allowed_ip=%s\n", cidr)
		}
		if ka := p.PersistentKeepalive; ka > 0 {
			fmt.Fprintf(&b, "persistent_keepalive_interval=%d\n", ka)
		}
	}
	// Blank line terminates the UAPI «set» operation (optional at EOF but matches tools).
	fmt.Fprintf(&b, "\n")
	return b.String(), nil
}

func splitAllowed(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{"0.0.0.0/0"}
	}
	raw := strings.Split(s, ",")
	res := make([]string, 0, len(raw))
	for _, x := range raw {
		if t := strings.TrimSpace(x); t != "" {
			res = append(res, t)
		}
	}
	if len(res) == 0 {
		res = append(res, "0.0.0.0/0")
	}
	return res
}

// startUAPI is platform-specific. On Linux it exposes the userspace
// wireguard-go device through /var/run/wireguard/<tunName>.sock so that
// wgctrl (and `wg show`) can read live peer byte counters and last-handshake
// timestamps. Without it, the stats collector silently sees nothing and the
// Statistics / Tunnels pages stay frozen. See manager_uapi_linux.go.

func (m *Manager) Create(tunnelID int, mtu int, listenPort int, privB64 string, peers []*domain.WgPeer, pskForPub func(pub string) []byte, tunName string) error {
	logger := device.NewLogger(device.LogLevelSilent, "") // avoid noisy WG logs alongside zerolog

	tunDev, err := tun.CreateTUN(tunName, mtu)
	if err != nil {
		return err
	}

	dev := device.NewDevice(tunDev, conn.NewDefaultBind(), logger)
	cfg, err := buildIPC(listenPort, privB64, peers, pskForPub)
	if err != nil {
		dev.Close()
		return err
	}
	if err := dev.IpcSet(cfg); err != nil {
		dev.Close()
		log.Error().Int("tunnel", tunnelID).Str("tun", tunName).Err(err).Msg("tunnel_trace wireguard: IpcSet failed")
		return err
	}

	uapi, uapiDone, err := startUAPI(dev, tunName)
	if err != nil {
		// Stats won't work without UAPI, but we've already brought the
		// dataplane up — don't kill the tunnel just because /var/run is
		// unwritable; just log and continue. Operators see the warning
		// and can mount /var/run if they want stats.
		log.Warn().Int("tunnel", tunnelID).Str("tun", tunName).Err(err).Msg("tunnel_trace wireguard: UAPI listen failed — live stats will be unavailable")
	} else {
		log.Info().Str("tun", tunName).Msg("tunnel_trace wireguard: UAPI listener started")
	}

	log.Info().
		Str("tun", tunName).
		Int("tunnel", tunnelID).
		Int("listen_port", listenPort).
		Int("mtu", mtu).
		Int("peer_count", len(peers)).
		Msg("tunnel_trace wireguard: IpcSet ok, device up")

	m.mu.Lock()
	defer m.mu.Unlock()
	if prev := m.devices[tunnelID]; prev != nil {
		if prev.uapi != nil {
			_ = prev.uapi.Close()
		}
		prev.dev.Close()
	}
	m.devices[tunnelID] = &managedDevice{dev: dev, uapi: uapi, uapiDone: uapiDone}
	return nil
}

func (m *Manager) ReloadPeers(tunnelID int, mtu int, listenPort int, privB64 string, peers []*domain.WgPeer, pskForPub func(pub string) []byte) error {
	m.mu.RLock()
	md := m.devices[tunnelID]
	m.mu.RUnlock()
	if md == nil || md.dev == nil {
		return fmt.Errorf("wg device missing for tunnel %d", tunnelID)
	}
	cfg, err := buildIPC(listenPort, privB64, peers, pskForPub)
	if err != nil {
		return err
	}
	return md.dev.IpcSet(cfg)
}

func (m *Manager) Destroy(tunnelID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	md := m.devices[tunnelID]
	if md == nil {
		return
	}
	delete(m.devices, tunnelID)
	if md.uapi != nil {
		_ = md.uapi.Close()
	}
	md.dev.Close()
}

func (m *Manager) Exists(tunnelID int) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.devices[tunnelID]
	return ok
}

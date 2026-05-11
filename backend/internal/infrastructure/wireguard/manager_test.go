package wireguardinfra

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"xray2wg/backend/internal/domain"

	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/tuntest"
	"golang.zx2c4.com/wireguard/wgctrl"
)

func TestBuildIPC_HasReplacePeersAndAllowedIP(t *testing.T) {
	priv := "kKp+Vf6dPjJgcQiTRplBlYwJj4MePeAQ8WzwzTPo718="
	peers := []*domain.WgPeer{
		{
			PublicKey:           "Q3Q3Q3Q3Q3Q3Q3Q3Q3Q3Q3Q3Q3Q3Q3Q3Q3Q3Q3Q3Q3A=",
			AllowedIPs:          "10.100.5.2/32",
			PersistentKeepalive: 25,
		},
	}
	cfg, err := buildIPC(51820, priv, peers, nil)
	if err != nil {
		t.Fatalf("buildIPC: %v", err)
	}
	for _, want := range []string{
		"private_key=",
		"listen_port=51820",
		"replace_peers=true",
		"public_key=",
		"replace_allowed_ips=true",
		"allowed_ip=10.100.5.2/32",
		"persistent_keepalive_interval=25",
	} {
		if !strings.Contains(cfg, want) {
			t.Fatalf("buildIPC missing %q\n%s", want, cfg)
		}
	}
}

func TestSplitAllowed_DefaultsToCatchAll(t *testing.T) {
	got := splitAllowed("")
	if len(got) != 1 || got[0] != "0.0.0.0/0" {
		t.Fatalf("empty splitAllowed = %v, want [0.0.0.0/0]", got)
	}
	got = splitAllowed("10.0.0.0/8, 192.168.0.0/16 ,, fd00::/8")
	want := []string{"10.0.0.0/8", "192.168.0.0/16", "fd00::/8"}
	if len(got) != len(want) {
		t.Fatalf("splitAllowed = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("splitAllowed[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestStartUAPI_ExposesDeviceToWgctrl is the regression guard for the
// "live stats not displayed" bug: without a UAPI socket, wgctrl cannot
// see a userspace wireguard-go device, so the stats collector silently
// drops every tick. We mock the TUN with tuntest, call startUAPI, and
// confirm wgctrl can list and fetch the device.
//
// Requires Linux + permission to create a unix socket in /var/run/wireguard,
// so it skips on Windows/macOS dev machines and on unprivileged CI. The
// production container runs with the necessary capabilities.
func TestStartUAPI_ExposesDeviceToWgctrl(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("skipping: requires linux UAPI socket support, got %s", runtime.GOOS)
	}
	if err := os.MkdirAll("/var/run/wireguard", 0o755); err != nil {
		t.Skipf("skipping: cannot create /var/run/wireguard (need root or capability): %v", err)
	}
	probe, err := os.CreateTemp("/var/run/wireguard", "x2w-probe-*")
	if err != nil {
		t.Skipf("skipping: /var/run/wireguard not writable (%v)", err)
	}
	_ = probe.Close()
	_ = os.Remove(probe.Name())

	tunName := "tun-wgtest"
	chtun := tuntest.NewChannelTUN()
	dev := device.NewDevice(chtun.TUN(), nil, device.NewLogger(device.LogLevelSilent, ""))
	defer dev.Close()

	uapi, _, err := startUAPI(dev, tunName)
	if err != nil {
		t.Fatalf("startUAPI: %v", err)
	}
	defer uapi.Close()

	// Give the accept loop a beat to register before wgctrl dials.
	time.Sleep(50 * time.Millisecond)

	c, err := wgctrl.New()
	if err != nil {
		t.Fatalf("wgctrl.New: %v", err)
	}
	defer c.Close()

	d, err := c.Device(tunName)
	if err != nil {
		t.Fatalf("wgctrl.Device(%s): %v — UAPI listener is not exposing the device", tunName, err)
	}
	if d.Name != tunName {
		t.Fatalf("wgctrl returned device %q, want %q", d.Name, tunName)
	}
}

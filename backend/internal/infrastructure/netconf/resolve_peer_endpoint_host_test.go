package netconf

import (
	"net"
	"testing"
)

func Test_resolvePeerEndpointHostFromSnapshots_priority(t *testing.T) {
	snaps := []ifaceAddrSnapshot{
		{Name: "eth0", Flags: net.FlagUp, IPs: []net.IP{net.ParseIP("192.168.31.28").To4()}},
	}
	t.Run("server_host wins over public and candidates", func(t *testing.T) {
		got := resolvePeerEndpointHostFromSnapshots("wg.example.com", "PUBLIC", false, snaps)
		if got != "wg.example.com" {
			t.Fatalf("got %q want wg.example.com", got)
		}
	})
	t.Run("public env when server empty", func(t *testing.T) {
		got := resolvePeerEndpointHostFromSnapshots("", "PUBLIC", false, snaps)
		if got != "PUBLIC" {
			t.Fatalf("got %q want PUBLIC", got)
		}
	})
	t.Run("detects LAN when no explicit host", func(t *testing.T) {
		got := resolvePeerEndpointHostFromSnapshots("", "", false, snaps)
		if got != "192.168.31.28" {
			t.Fatalf("got %q want 192.168.31.28", got)
		}
	})
	t.Run("empty host regression never returns bare empty", func(t *testing.T) {
		got := resolvePeerEndpointHostFromSnapshots("", "", false, nil)
		if got == "" {
			t.Fatal("expected non-empty placeholder")
		}
		if got != UnconfiguredPeerEndpointHost {
			t.Fatalf("got %q want %q", got, UnconfiguredPeerEndpointHost)
		}
	})
}

func Test_pickPeerEndpointHostFromSnapshots_docker172_17(t *testing.T) {
	snaps := []ifaceAddrSnapshot{
		{Name: "eth0", Flags: net.FlagUp, IPs: []net.IP{net.ParseIP("172.17.0.2").To4()}},
	}
	got := pickPeerEndpointHostFromSnapshots(true, snaps)
	if got != "" {
		t.Fatalf("expected skip of 172.17 in docker, got %q", got)
	}
}

func Test_pickPeerEndpointHostFromSnapshots_prefers192_168(t *testing.T) {
	snaps := []ifaceAddrSnapshot{
		{Name: "eth0", Flags: net.FlagUp, IPs: []net.IP{
			net.ParseIP("10.0.0.5").To4(),
			net.ParseIP("192.168.31.28").To4(),
		}},
	}
	got := pickPeerEndpointHostFromSnapshots(false, snaps)
	if got != "192.168.31.28" {
		t.Fatalf("got %q want 192.168.31.28", got)
	}
}

func Test_skipInterfaceNameForPeerEndpoint(t *testing.T) {
	for _, n := range []string{"docker0", "br-abc123", "veth0", "tun-wg1", "eth0"} {
		skip := skipInterfaceNameForPeerEndpoint(n)
		want := n != "eth0"
		if skip != want {
			t.Fatalf("%q: skip=%v want %v", n, skip, want)
		}
	}
}

func Test_pickPeerEndpointHostFromSnapshots_publicIPv4(t *testing.T) {
	snaps := []ifaceAddrSnapshot{
		{Name: "eth0", Flags: net.FlagUp, IPs: []net.IP{net.ParseIP("203.0.113.5").To4()}},
	}
	got := pickPeerEndpointHostFromSnapshots(false, snaps)
	if got != "203.0.113.5" {
		t.Fatalf("got %q want 203.0.113.5", got)
	}
}

func Test_resolvePeerEndpointHostFromSnapshots_skips_docker_named(t *testing.T) {
	snaps := []ifaceAddrSnapshot{
		{Name: "docker0", Flags: net.FlagUp, IPs: []net.IP{net.ParseIP("172.17.0.1").To4()}},
		{Name: "enp0s1", Flags: net.FlagUp, IPs: []net.IP{net.ParseIP("192.168.1.10").To4()}},
	}
	got := resolvePeerEndpointHostFromSnapshots("", "", false, snaps)
	if got != "192.168.1.10" {
		t.Fatalf("got %q want 192.168.1.10", got)
	}
}

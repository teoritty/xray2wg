package service

import (
	"strings"
	"testing"
)

func Test_wireGuardPeerClientIni_omitsEmptyPSKAndDNS(t *testing.T) {
	got := wireGuardPeerClientIni(
		"aPrivateKey=",
		"10.0.0.2/32",
		"",
		1420,
		"bServerPub=",
		"",
		"0.0.0.0/0",
		25,
		51820,
	)
	if strings.Contains(got, "PresharedKey") {
		t.Fatalf("expected empty PSK to omit PresharedKey line, got:\n%s", got)
	}
	if strings.Contains(got, "DNS =") {
		t.Fatalf("expected empty DNS to omit DNS line, got:\n%s", got)
	}
	if !strings.Contains(got, "Endpoint = "+WgPeerConfigEndpointHostPlaceholder+":51820") {
		t.Fatalf("missing endpoint placeholder, got:\n%s", got)
	}
}

func Test_wireGuardPeerClientIni_includesPSKAndDNSWhenSet(t *testing.T) {
	got := wireGuardPeerClientIni(
		"priv=",
		"10.0.0.3/32",
		"1.1.1.1",
		1280,
		"pub=",
		"psk=",
		"::/0,0.0.0.0/0",
		0,
		1194,
	)
	if !strings.Contains(got, "DNS = 1.1.1.1") {
		t.Fatalf("expected DNS line, got:\n%s", got)
	}
	if !strings.Contains(got, "PresharedKey = psk=") {
		t.Fatalf("expected PresharedKey line, got:\n%s", got)
	}
	if !strings.Contains(got, "PersistentKeepalive = 0") {
		t.Fatalf("expected PersistentKeepalive line, got:\n%s", got)
	}
}

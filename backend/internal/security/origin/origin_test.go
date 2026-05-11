package origin

import (
	"sync"
	"testing"
)

func TestAllowOriginExactMatch(t *testing.T) {
	cfg, err := NewConfig("https://vpn.example.com,http://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowOrigin("https://vpn.example.com") {
		t.Fatal("expected exact https match")
	}
	if !cfg.AllowOrigin("HTTPS://VPN.EXAMPLE.COM") {
		t.Fatal("expected case-insensitive scheme/host normalization")
	}
	if !cfg.AllowOrigin("http://localhost:3000") {
		t.Fatal("expected second allowed origin")
	}
}

func TestSuffixAttackNotSubstring(t *testing.T) {
	cfg, err := NewConfig("https://vpn.example.com")
	if err != nil {
		t.Fatal(err)
	}
	evil := "https://evil-vpn.example.com"
	if cfg.AllowOrigin(evil) {
		t.Fatalf("suffix-style host must not match: %q", evil)
	}
}

func TestPrefixSubdomainAttack(t *testing.T) {
	cfg, err := NewConfig("https://vpn.example.com")
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range []string{
		"https://sub.vpn.example.com",
		"https://vpn.example.com.evil.net",
		"https://notvpn.example.com",
	} {
		if cfg.AllowOrigin(o) {
			t.Fatalf("unexpected allow for %q", o)
		}
	}
}

func TestPortMismatch8080vs9090(t *testing.T) {
	cfg, err := NewConfig("https://vpn.example.com:8080")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowOrigin("https://vpn.example.com:8080") {
		t.Fatal("expected allow exact port 8080")
	}
	if cfg.AllowOrigin("https://vpn.example.com:9090") {
		t.Fatal("different port must not match")
	}
}

func TestHttpsDefaultPortVsExplicit443(t *testing.T) {
	cfg, err := NewConfig("https://vpn.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.AllowOrigin("https://vpn.example.com") {
		t.Fatal("expected allow without explicit port")
	}
	if cfg.AllowOrigin("https://vpn.example.com:443") {
		t.Fatal("strict: explicit :443 must not match implicit default (different origin string)")
	}
}

func TestSchemeMismatch(t *testing.T) {
	cfg, err := NewConfig("https://vpn.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AllowOrigin("http://vpn.example.com") {
		t.Fatal("http must not match https allowed origin")
	}
}

func TestSubstringInjection(t *testing.T) {
	cfg, err := NewConfig("https://vpn.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AllowOrigin("https://xhttps://vpn.example.com") {
		t.Fatal("malformed origin must not substring-match")
	}
}

func TestEmptyOriginRejected(t *testing.T) {
	cfg, err := NewConfig("https://a.example")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AllowOrigin("") {
		t.Fatal("empty origin must be rejected")
	}
	if cfg.AllowOrigin("   ") {
		t.Fatal("whitespace-only origin must be rejected")
	}
}

func TestNewConfigEmptyRejected(t *testing.T) {
	for _, in := range []string{"", "  ", ",", " , , "} {
		_, err := NewConfig(in)
		if err == nil {
			t.Fatalf("expected error for %q", in)
		}
	}
}

func TestConcurrentAccessNoRace(t *testing.T) {
	cfg, err := NewConfig("https://vpn.example.com,https://other.example:8443")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = cfg.AllowOrigin("https://vpn.example.com")
				_ = cfg.AllowOrigin("https://evil.example")
				_ = cfg.AllowedOrigins()
			}
		}()
	}
	wg.Wait()
}

func TestAllowedOriginsSortedCopy(t *testing.T) {
	cfg, err := NewConfig("https://z.example,https://a.example")
	if err != nil {
		t.Fatal(err)
	}
	a := cfg.AllowedOrigins()
	if len(a) != 2 || a[0] != "https://a.example" || a[1] != "https://z.example" {
		t.Fatalf("got %v", a)
	}
	a[0] = "mutated"
	b := cfg.AllowedOrigins()
	if b[0] != "https://a.example" {
		t.Fatal("internal set must not be mutated via returned slice")
	}
}

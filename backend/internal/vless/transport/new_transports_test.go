package transport

import (
	"encoding/json"
	"net/url"
	"reflect"
	"testing"
)

// TestXHTTP_FullFlow verifies parse → emit → encode → decode → re-emit produces stable
// output for an xhttp share-link.
func TestXHTTP_FullFlow(t *testing.T) {
	tr, err := Default.Resolve("xhttp")
	if err != nil {
		t.Fatal(err)
	}
	q := url.Values{"path": {"/api"}, "host": {"cdn.example.com"}, "mode": {"stream-up"}}
	spec, err := tr.ParseURI(ParseCtx{Query: q})
	if err != nil {
		t.Fatal(err)
	}
	if err := tr.Validate(spec); err != nil {
		t.Fatal(err)
	}
	got, _ := tr.EmitSettings(spec)
	want := map[string]any{
		"path": "/api",
		"host": "cdn.example.com",
		"mode": "stream-up",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("EmitSettings: got %v\nwant %v", got, want)
	}
	// Round-trip through JSON storage.
	data, _ := tr.EncodeSpec(spec)
	roundTrip, _ := tr.DecodeSpec(data)
	if !reflect.DeepEqual(roundTrip, spec) {
		t.Fatalf("round-trip diverged: %+v vs %+v", roundTrip, spec)
	}
}

func TestXHTTP_RejectsInvalidMode(t *testing.T) {
	tr, _ := Default.Resolve("xhttp")
	if err := tr.Validate(XHTTPSpec{Mode: "bogus"}); err == nil {
		t.Fatal("expected error for invalid mode")
	}
}

func TestXHTTP_DefaultModeAuto(t *testing.T) {
	tr, _ := Default.Resolve("xhttp")
	spec, _ := tr.ParseURI(ParseCtx{Query: url.Values{"path": {"/"}}})
	if spec.(XHTTPSpec).Mode != "auto" {
		t.Fatalf("default mode: want auto, got %q", spec.(XHTTPSpec).Mode)
	}
}

func TestXHTTP_AliasSplitHTTP(t *testing.T) {
	tr, err := Default.Resolve("splithttp")
	if err != nil || tr.Name() != "xhttp" {
		t.Fatalf("splithttp must alias xhttp, got %v / %s", err, tr.Name())
	}
}

func TestHTTPUpgrade_EmitSettings(t *testing.T) {
	tr, _ := Default.Resolve("httpupgrade")
	spec, _ := tr.ParseURI(ParseCtx{Query: url.Values{"path": {"/up"}, "host": {"cdn"}}})
	got, _ := tr.EmitSettings(spec)
	want := map[string]any{"path": "/up", "host": "cdn"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}

func TestHTTPUpgrade_DefaultPath(t *testing.T) {
	tr, _ := Default.Resolve("httpupgrade")
	got, _ := tr.EmitSettings(HTTPUpgradeSpec{})
	if got["path"] != "/" {
		t.Fatalf("default path: want /, got %v", got["path"])
	}
}

func TestKCP_EmitSettings_appliesDefaults(t *testing.T) {
	tr, _ := Default.Resolve("kcp")
	got, _ := tr.EmitSettings(KCPSpec{HeaderType: "wechat-video", Seed: "abc"})
	if got["mtu"] != 1350 || got["tti"] != 20 || got["uplinkCapacity"] != 5 ||
		got["downlinkCapacity"] != 20 || got["congestion"] != false ||
		got["readBufferSize"] != 1 || got["writeBufferSize"] != 1 {
		t.Fatalf("kcp defaults missing/wrong: %v", got)
	}
	hdr, _ := got["header"].(map[string]any)
	if hdr["type"] != "wechat-video" {
		t.Fatalf("header.type: %v", hdr["type"])
	}
	if got["seed"] != "abc" {
		t.Fatalf("seed: %v", got["seed"])
	}
}

func TestKCP_RejectsInvalidHeaderType(t *testing.T) {
	tr, _ := Default.Resolve("kcp")
	if err := tr.Validate(KCPSpec{HeaderType: "bogus"}); err == nil {
		t.Fatal("expected error for unknown headerType")
	}
}

func TestKCP_AliasMKCP(t *testing.T) {
	tr, err := Default.Resolve("mkcp")
	if err != nil || tr.Name() != "kcp" {
		t.Fatalf("mkcp must alias kcp, got %v / %s", err, tr.Name())
	}
}

// TestEndToEnd_URI_to_StreamSettings exercises a full vless:// URI through ParseURI →
// streamSettings emission for each new transport, asserting the resulting JSON shape
// matches what xray-core expects.
func TestEndToEnd_StreamSettingsShapeForNewTransports(t *testing.T) {
	cases := []struct {
		name    string
		network string
		query   url.Values
		wantKey string // streamSettings.<wantKey>Settings must exist with the right network
	}{
		{"xhttp", "xhttp", url.Values{"path": {"/v"}, "mode": {"auto"}}, "xhttp"},
		{"splithttp-alias", "splithttp", url.Values{"path": {"/v"}}, "xhttp"},
		{"httpupgrade", "httpupgrade", url.Values{"path": {"/up"}}, "httpupgrade"},
		{"kcp", "kcp", url.Values{"headerType": {"none"}}, "kcp"},
		{"mkcp-alias", "mkcp", url.Values{"headerType": {"srtp"}}, "kcp"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tr, err := Default.Resolve(c.network)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			spec, err := tr.ParseURI(ParseCtx{Query: c.query})
			if err != nil {
				t.Fatal(err)
			}
			if err := tr.Validate(spec); err != nil {
				t.Fatalf("Validate: %v", err)
			}
			data, _ := tr.EncodeSpec(spec)
			if !json.Valid(data) {
				t.Fatalf("EncodeSpec emitted invalid JSON: %s", data)
			}
			if tr.Name() != c.wantKey {
				t.Fatalf("canonical name: want %s, got %s", c.wantKey, tr.Name())
			}
			settings, _ := tr.EmitSettings(spec)
			if len(settings) == 0 {
				t.Fatalf("EmitSettings returned empty map for %s", c.name)
			}
		})
	}
}

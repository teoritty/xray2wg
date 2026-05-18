package security

import (
	"net/url"
	"reflect"
	"testing"
)

func TestReality_EmitSettings_appliesDefaults(t *testing.T) {
	sec, _ := Default.Resolve("reality")
	got, _ := sec.EmitSettings(RealitySpec{
		ServerName: "ex.com",
		PublicKey:  "PBK",
		ShortID:    "ab",
	})
	want := map[string]any{
		"fingerprint": "chrome",
		"serverName":  "ex.com",
		"publicKey":   "PBK",
		"shortId":     "ab",
		"spiderX":     "/",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}

func TestReality_ParseURI(t *testing.T) {
	sec, _ := Default.Resolve("reality")
	q := url.Values{"sni": {"a"}, "fp": {"firefox"}, "pbk": {"P"}, "sid": {"S"}, "spx": {"/x"}}
	spec, _ := sec.ParseURI(ParseCtx{Query: q})
	got := spec.(RealitySpec)
	want := RealitySpec{ServerName: "a", Fingerprint: "firefox", PublicKey: "P", ShortID: "S", SpiderX: "/x"}
	if got != want {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
}

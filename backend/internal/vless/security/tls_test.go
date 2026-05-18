package security

import (
	"net/url"
	"reflect"
	"testing"
)

func TestTLS_EmitSettings_alpnArrayShape(t *testing.T) {
	sec, _ := Default.Resolve("tls")
	got, _ := sec.EmitSettings(TLSSpec{ServerName: "ex.com", ALPN: []string{"h2", "http/1.1"}})
	want := map[string]any{
		"allowInsecure": false,
		"serverName":    "ex.com",
		"alpn":          []any{"h2", "http/1.1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v\nwant %v", got, want)
	}
}

func TestTLS_EmitSettings_emptyAlpnSerializesAsNull(t *testing.T) {
	sec, _ := Default.Resolve("tls")
	got, _ := sec.EmitSettings(TLSSpec{ServerName: "ex.com"})
	v, ok := got["alpn"].([]any)
	if ok && len(v) != 0 {
		t.Fatalf("empty alpn must marshal as null or empty array, got %v", v)
	}
}

func TestTLS_ParseURI_alpnSplit(t *testing.T) {
	sec, _ := Default.Resolve("tls")
	q := url.Values{"alpn": {"h2, http/1.1 ,  "}, "sni": {"x"}, "fp": {"chrome"}}
	spec, _ := sec.ParseURI(ParseCtx{Query: q})
	got := spec.(TLSSpec)
	if !reflect.DeepEqual(got.ALPN, []string{"h2", "http/1.1"}) {
		t.Fatalf("alpn split: %v", got.ALPN)
	}
}

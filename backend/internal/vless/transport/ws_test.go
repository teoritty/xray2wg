package transport

import (
	"net/url"
	"reflect"
	"testing"

	"xray2wg/backend/internal/domain"
)

func TestWS_EmitSettings_defaultPath(t *testing.T) {
	tr, _ := Default.Resolve("ws")
	settings, err := tr.EmitSettings(WSSpec{Path: "", Host: "cdn.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"path":    "/",
		"headers": map[string]any{"Host": "cdn.example.com"},
	}
	if !reflect.DeepEqual(settings, want) {
		t.Fatalf("got %v\nwant %v", settings, want)
	}
}

func TestWS_EmitSettings_relativePathFallsBackToSlash(t *testing.T) {
	tr, _ := Default.Resolve("ws")
	settings, _ := tr.EmitSettings(WSSpec{Path: "relative", Host: "h"})
	if settings["path"] != "/" {
		t.Fatalf("relative path must fall back to /, got %q", settings["path"])
	}
}

func TestWS_ApplyToLegacyNode_writesPathToSpiderXAndHostToSNIWhenEmpty(t *testing.T) {
	tr, _ := Default.Resolve("ws")
	n := &domain.VlessNode{}
	tr.ApplyToLegacyNode(WSSpec{Path: "/v", Host: "cdn.example"}, n)
	if n.SpiderX != "/v" || n.SNI != "cdn.example" {
		t.Fatalf("ApplyToLegacyNode: %+v", n)
	}
	// SNI must not be overwritten when already set.
	n2 := &domain.VlessNode{SNI: "explicit.sni"}
	tr.ApplyToLegacyNode(WSSpec{Path: "/v", Host: "cdn.example"}, n2)
	if n2.SNI != "explicit.sni" {
		t.Fatalf("WS must not overwrite an explicitly set SNI; got %q", n2.SNI)
	}
}

func TestWS_ParseURI(t *testing.T) {
	tr, _ := Default.Resolve("ws")
	q := url.Values{"path": {"/foo"}, "host": {"cdn"}}
	spec, err := tr.ParseURI(ParseCtx{Query: q})
	if err != nil {
		t.Fatal(err)
	}
	got := spec.(WSSpec)
	if got.Path != "/foo" || got.Host != "cdn" {
		t.Fatalf("ParseURI: %+v", got)
	}
}

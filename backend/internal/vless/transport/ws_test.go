package transport

import (
	"net/url"
	"reflect"
	"testing"
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

func TestWS_RoundTrip(t *testing.T) {
	tr, _ := Default.Resolve("ws")
	original := WSSpec{Path: "/v", Host: "cdn.example.com"}
	data, err := tr.EncodeSpec(original)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := tr.DecodeSpec(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf("round-trip: got %+v, want %+v", decoded, original)
	}
	share, _ := tr.ShareLink(original)
	if share.Get("path") != "/v" || share.Get("host") != "cdn.example.com" {
		t.Fatalf("ShareLink: %v", share)
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

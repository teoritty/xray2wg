package transport

import (
	"net/url"
	"reflect"
	"testing"
)

func TestGRPC_EmitSettings_defaultsGunService(t *testing.T) {
	tr, _ := Default.Resolve("grpc")
	settings, _ := tr.EmitSettings(GRPCSpec{})
	want := map[string]any{"serviceName": "GunService"}
	if !reflect.DeepEqual(settings, want) {
		t.Fatalf("got %v\nwant %v", settings, want)
	}
}

func TestGRPC_ParseURI_fallsBackToALPNForServiceName(t *testing.T) {
	tr, _ := Default.Resolve("grpc")
	q := url.Values{"alpn": {"my-svc"}}
	spec, _ := tr.ParseURI(ParseCtx{Query: q})
	got := spec.(GRPCSpec)
	if got.ServiceName != "my-svc" {
		t.Fatalf("alpn fallback: %+v", got)
	}
}

func TestGRPC_ParseURI_serviceNamePrefersExplicit(t *testing.T) {
	tr, _ := Default.Resolve("grpc")
	q := url.Values{"alpn": {"x"}, "serviceName": {"y"}}
	spec, _ := tr.ParseURI(ParseCtx{Query: q})
	got := spec.(GRPCSpec)
	if got.ServiceName != "y" {
		t.Fatalf("explicit serviceName must win: %+v", got)
	}
}

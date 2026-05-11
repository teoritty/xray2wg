package netconf

import "testing"

func TestWgGatewayHost(t *testing.T) {
	h, err := WgGatewayHost("10.100.3.1/24")
	if err != nil {
		t.Fatal(err)
	}
	if h != "10.100.3.1" {
		t.Fatalf("got %q", h)
	}
}

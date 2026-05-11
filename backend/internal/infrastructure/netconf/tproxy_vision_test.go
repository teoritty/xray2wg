//go:build linux

package netconf

import "testing"

func TestDropQUICVision(t *testing.T) {
	tests := []struct {
		flow string
		want bool
	}{
		{"xtls-rprx-vision", true},
		{"XTLS-RPRX-VISION", true},
		{"xtls-rprx-vision-udp443", false},
		{"", false},
		{"none", false},
	}
	for _, tc := range tests {
		if got := dropQUICVision(tc.flow); got != tc.want {
			t.Fatalf("dropQUICVision(%q) = %v, want %v", tc.flow, got, tc.want)
		}
	}
}

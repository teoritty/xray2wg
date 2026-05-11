package infra

import (
	"testing"

	"xray2wg/backend/internal/service"
)

// Interface satisfaction at compile time.
var (
	_ service.XrayProcess = (*XrayAdapter)(nil)
	_ service.WgTunnel    = (*WgAdapter)(nil)
)

func TestAdapterInterfaceSatisfaction(_ *testing.T) {}

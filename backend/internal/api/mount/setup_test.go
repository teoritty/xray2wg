package mount

import (
	"os"
	"testing"

	"xray2wg/backend/internal/telemetry"

	"github.com/prometheus/client_golang/prometheus"
)

func TestMain(m *testing.M) {
	telemetry.Register(prometheus.DefaultRegisterer)
	os.Exit(m.Run())
}

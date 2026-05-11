package wireguardinfra

import (
	"context"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
)

type PeerStats struct {
	PublicKey     string
	RxBytes       int64
	TxBytes       int64
	LastHandshake time.Time
}

func PollStats(ctx context.Context, tunName string) ([]PeerStats, error) {
	_ = ctx
	c, err := wgctrl.New()
	if err != nil {
		return nil, err
	}
	defer c.Close()
	dev, err := c.Device(tunName)
	if err != nil {
		return nil, err
	}
	var out []PeerStats
	for _, p := range dev.Peers {
		out = append(out, PeerStats{
			PublicKey:     p.PublicKey.String(),
			RxBytes:       p.ReceiveBytes,
			TxBytes:       p.TransmitBytes,
			LastHandshake: p.LastHandshakeTime,
		})
	}
	return out, nil
}

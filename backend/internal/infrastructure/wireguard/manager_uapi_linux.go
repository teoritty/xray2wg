//go:build linux

package wireguardinfra

import (
	"fmt"
	"net"

	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
)

// startUAPI exposes the userspace wireguard-go device through the standard
// /var/run/wireguard/<tunName>.sock UAPI socket. wgctrl (used by the stats
// collector and `wg show`) connects there to read live byte counters and
// last-handshake timestamps. Without this socket, PollStats fails on every
// tick and the live RX/TX rates on /#/tunnels and /#/statistics stay at 0.
func startUAPI(dev *device.Device, tunName string) (net.Listener, chan struct{}, error) {
	fileUAPI, err := ipc.UAPIOpen(tunName)
	if err != nil {
		return nil, nil, fmt.Errorf("uapi open: %w", err)
	}
	uapi, err := ipc.UAPIListen(tunName, fileUAPI)
	if err != nil {
		_ = fileUAPI.Close()
		return nil, nil, fmt.Errorf("uapi listen: %w", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			c, err := uapi.Accept()
			if err != nil {
				return
			}
			go dev.IpcHandle(c)
		}
	}()
	return uapi, done, nil
}

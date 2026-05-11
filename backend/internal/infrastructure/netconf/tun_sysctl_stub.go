//go:build !linux

package netconf

func PrepareTunForTransparentProxy(string) {}

func LogIPv4ForwardingSnapshot(string) {}

#!/bin/sh
set -e

# Writable /proc/sys is not guaranteed (read-only rootfs, some k8s setups). Docker compose
# often sets net.ipv4.ip_forward in the container namespace instead.
if [ -w /proc/sys/net/ipv4/ip_forward ]; then
	sysctl -w net.ipv4.ip_forward=1 >/dev/null 2>&1 || true
fi
if [ -w /proc/sys/net/ipv6/conf/all/forwarding ]; then
	sysctl -w net.ipv6.conf.all.forwarding=1 >/dev/null 2>&1 || true
fi

iptables -t mangle -F 2>/dev/null || true
iptables -t mangle -X 2>/dev/null || true

exec /usr/local/bin/xray2wg "$@"

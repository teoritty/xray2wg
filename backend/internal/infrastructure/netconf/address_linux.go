//go:build linux

package netconf

import (
	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
)

func AssignTUN(tunName, cidr string) error {
	link, err := netlink.LinkByName(tunName)
	if err != nil {
		return err
	}
	addr, err := netlink.ParseAddr(cidr)
	if err != nil {
		return err
	}
	if err := netlink.AddrReplace(link, addr); err != nil {
		log.Warn().Err(err).Str("dev", tunName).Msg("tunnel_trace assign_tun: AddrReplace warn")
	} else {
		log.Info().Str("dev", tunName).Str("cidr", cidr).Msg("tunnel_trace assign_tun: AddrReplace ok")
	}
	if err := netlink.LinkSetUp(link); err != nil {
		log.Error().Err(err).Str("dev", tunName).Msg("tunnel_trace assign_tun: LinkSetUp failed")
		return err
	}
	log.Info().Str("dev", tunName).Msg("tunnel_trace assign_tun: LinkSetUp ok")
	return nil
}

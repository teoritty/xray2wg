//go:build linux

package netconf

import (
	"fmt"
	"net"

	"xray2wg/backend/internal/tracenv"

	"github.com/rs/zerolog/log"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func SetupReturnRoute(fwmark int, routingTable int) error {
	log.Info().Int("fwmark", fwmark).Int("routing_table", routingTable).Msg("tunnel_trace policy_route: SetupReturnRoute begin")
	fw := uint32(fwmark)
	mask := uint32(0xffffffff)

	rule := netlink.NewRule()
	rule.Mark = fw
	rule.Mask = &mask
	rule.Table = routingTable
	rule.Priority = 32765
	rule.Invert = false
	if err := netlink.RuleAdd(rule); err != nil {
		log.Warn().Err(err).Uint32("mark", fw).Msg("tunnel_trace policy_route: RuleAdd (may already exist)")
	} else {
		log.Info().Uint32("mark", fw).Int("table", routingTable).Msg("tunnel_trace policy_route: ip rule added")
	}

	lo, err := netlink.LinkByName("lo")
	if err != nil {
		log.Error().Err(err).Msg("tunnel_trace policy_route: LinkByName lo failed")
		return err
	}
	r := netlink.Route{
		Scope: netlink.SCOPE_HOST,
		Dst: &net.IPNet{
			IP:   net.IPv4(0, 0, 0, 0),
			Mask: net.CIDRMask(0, 32),
		},
		Type:      unix.RTN_LOCAL,
		LinkIndex: lo.Attrs().Index,
		Family:    unix.AF_INET,
		Table:     routingTable,
		Protocol:  unix.RTPROT_KERNEL,
	}
	if err := netlink.RouteReplace(&r); err != nil {
		log.Error().Int("table", routingTable).Err(err).Msg("tunnel_trace policy_route: RouteReplace failed")
		return fmt.Errorf("route replace table=%d: %w", routingTable, err)
	}
	log.Info().Int("table", routingTable).Str("type", "RTN_LOCAL").Str("dev", "lo").Msg("tunnel_trace policy_route: default route in table set (tproxy return path)")

	if tracenv.TunnelTrace() {
		rules, err := netlink.RuleList(netlink.FAMILY_V4)
		if err != nil {
			log.Warn().Err(err).Msg("tunnel_trace policy_route: RuleList failed")
		} else {
			var hits []string
			for _, ru := range rules {
				if ru.Table == routingTable {
					hits = append(hits, fmt.Sprintf("prio=%d mark=%#x table=%d", ru.Priority, ru.Mark, ru.Table))
				}
			}
			log.Info().Strs("matching_rules", hits).Msg("tunnel_trace policy_route: rules using this table")
		}
	}

	return nil
}

func TeardownReturnRoute(fwmark int, routingTable int) error {
	log.Info().Int("fwmark", fwmark).Int("routing_table", routingTable).Msg("tunnel_trace policy_route: TeardownReturnRoute")
	fw := uint32(fwmark)
	rules, err := netlink.RuleList(netlink.FAMILY_V4)
	if err != nil {
		return err
	}
	for _, ru := range rules {
		if ru.Mark == fw && ru.Table == routingTable {
			if err := netlink.RuleDel(&ru); err != nil {
				log.Warn().Err(err).Msg("rule del")
			}
		}
	}
	rs, err := netlink.RouteListFiltered(netlink.FAMILY_V4, &netlink.Route{Table: routingTable}, netlink.RT_FILTER_TABLE)
	if err != nil {
		return err
	}
	for i := range rs {
		_ = netlink.RouteDel(&rs[i])
	}
	return nil
}

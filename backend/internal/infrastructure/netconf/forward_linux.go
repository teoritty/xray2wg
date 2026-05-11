//go:build linux

package netconf

import (
	"github.com/coreos/go-iptables/iptables"
	"github.com/rs/zerolog/log"
)

const tableFilter = "filter"

const chainDockerUser = "DOCKER-USER"

// SetupForwardRules allows IPv4 forwarding between the WG TUN and the rest of the namespace.
// Use DOCKER-USER with insert-at-head when present (Docker FORWARD often DROPs before appended rules).
func SetupForwardRules(tunName string) error {
	log.Info().Str("tun", tunName).Msg("tunnel_trace forward: SetupForwardRules begin")
	tb, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv4))
	if err != nil {
		log.Error().Err(err).Msg("tunnel_trace forward: iptables.New failed")
		return err
	}
	teardownForwardRules(tb, tunName)

	useDockerUser := false
	if ok, _ := tb.ChainExists(tableFilter, chainDockerUser); ok {
		eOut := tb.Insert(tableFilter, chainDockerUser, 1, "-o", tunName, "-j", "ACCEPT")
		eIn := tb.Insert(tableFilter, chainDockerUser, 1, "-i", tunName, "-j", "ACCEPT")
		if eOut == nil && eIn == nil {
			useDockerUser = true
			log.Info().Str("tun", tunName).Msg("tunnel_trace forward: DOCKER-USER ACCEPT -i/-o installed (position 1)")
		} else {
			_ = tb.Delete(tableFilter, chainDockerUser, "-o", tunName, "-j", "ACCEPT")
			_ = tb.Delete(tableFilter, chainDockerUser, "-i", tunName, "-j", "ACCEPT")
			log.Warn().Err(eOut).AnErr("insert_i", eIn).Str("tun", tunName).Msg("tunnel_trace forward: DOCKER-USER insert failed, will use FORWARD")
		}
	}
	if !useDockerUser {
		if err := tb.Insert(tableFilter, "FORWARD", 1, "-i", tunName, "-j", "ACCEPT"); err != nil {
			log.Error().Err(err).Msg("tunnel_trace forward: insert FORWARD -i tun failed")
			return err
		}
		if err := tb.Insert(tableFilter, "FORWARD", 1, "-o", tunName, "-j", "ACCEPT"); err != nil {
			_ = tb.Delete(tableFilter, "FORWARD", "-i", tunName, "-j", "ACCEPT")
			log.Error().Err(err).Msg("tunnel_trace forward: insert FORWARD -o tun failed")
			return err
		}
		log.Info().Str("tun", tunName).Msg("tunnel_trace forward: FORWARD ACCEPT -i/-o installed (position 1)")
	}
	return nil
}

func teardownForwardRules(tb *iptables.IPTables, tunName string) {
	for _, ch := range []string{"FORWARD", chainDockerUser} {
		_ = tb.Delete(tableFilter, ch, "-o", tunName, "-j", "ACCEPT")
		_ = tb.Delete(tableFilter, ch, "-i", tunName, "-j", "ACCEPT")
	}
}

func TeardownForwardRules(tunName string) error {
	log.Info().Str("tun", tunName).Msg("tunnel_trace forward: TeardownForwardRules")
	tb, err := iptables.New(iptables.IPFamily(iptables.ProtocolIPv4))
	if err != nil {
		return err
	}
	teardownForwardRules(tb, tunName)
	return nil
}

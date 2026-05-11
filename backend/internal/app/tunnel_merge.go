package app

import "xray2wg/backend/internal/domain"

// PreserveTunnelLifecycleOnMetadataUpdate keeps status/error/uptime owned by runtime.
func PreserveTunnelLifecycleOnMetadataUpdate(existing, incoming *domain.WgInterface) {
	if existing == nil || incoming == nil {
		return
	}
	incoming.Status = existing.Status
	incoming.ErrorMessage = existing.ErrorMessage
	incoming.UptimeStarted = existing.UptimeStarted
}

// PreserveTunnelUpdateBindings keeps active_node_id on partial subscription updates.
func PreserveTunnelUpdateBindings(existing, incoming *domain.WgInterface) {
	if existing == nil || incoming == nil || incoming.ActiveNodeID != nil || existing.ActiveNodeID == nil {
		return
	}
	incSub := incoming.SubscriptionID
	if incSub == nil {
		incSub = existing.SubscriptionID
	}
	if existing.SubscriptionID == nil || incSub == nil {
		return
	}
	if *existing.SubscriptionID == *incSub {
		incoming.ActiveNodeID = existing.ActiveNodeID
	}
}

package sqldb

import (
	"context"
	"path/filepath"
	"testing"

	"xray2wg/backend/internal/domain"

	gormsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestTunnelRepo_Update_preservesActiveNodeWhenNilInStruct(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(gormsqlite.Open("file:"+filepath.Join(t.TempDir(), "tunrepo.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		x, err := db.DB()
		if err == nil {
			_ = x.Close()
		}
	})
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}

	sub := SubscriptionRow{Name: "__manual__", URL: "", RefreshInterval: 86400, Status: string(domain.SubStatusInactive)}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	node := VlessNodeRow{
		SubscriptionID: sub.ID,
		DisplayName:    "n1",
		UUID:           "00000000-0000-0000-0000-000000000001",
		Address:        "example.com",
		Port:           443,
		RawURI:         "vless://00000000-0000-0000-0000-000000000001@example.com:443?encryption=none&security=reality&type=tcp#n1",
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}
	sid := sub.ID
	nid := node.ID
	row := WgInterfaceRow{
		Name:           "t1",
		PrivateKeyEnc:  "enc-placeholder-32bytes-min______",
		PublicKey:      "pubkey________________________________________=",
		ListenPort:     51820,
		WgAddress:      "10.100.1.1/24",
		SubscriptionID: &sid,
		ActiveNodeID:   &nid,
		Status:         string(domain.WgStatusStopped),
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewTunnelRepo(db)
	partial := &domain.WgInterface{
		ID:             row.ID,
		Name:           "t1",
		ListenPort:     51820,
		WgAddress:      "10.100.1.1/24",
		DNS:            "1.1.1.1",
		MTU:            1420,
		SubscriptionID: &sid,
		ActiveNodeID:   nil,
		Status:         domain.WgStatusRunning,
	}
	if err := repo.Update(ctx, partial); err != nil {
		t.Fatal(err)
	}

	var after WgInterfaceRow
	if err := db.First(&after, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.ActiveNodeID == nil || *after.ActiveNodeID != nid {
		t.Fatalf("active_node_id was cleared or wrong: %#v (want %d)", after.ActiveNodeID, nid)
	}
}

func TestTunnelRepo_ClearActiveNodeID(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(gormsqlite.Open("file:"+filepath.Join(t.TempDir(), "tunclr.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		x, err := db.DB()
		if err == nil {
			_ = x.Close()
		}
	})
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}

	sub := SubscriptionRow{Name: "s", URL: "http://x", RefreshInterval: 3600, Status: string(domain.SubStatusInactive)}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	node := VlessNodeRow{
		SubscriptionID: sub.ID,
		DisplayName:    "n1",
		UUID:           "00000000-0000-0000-0000-000000000002",
		Address:        "example.com",
		Port:           443,
		RawURI:         "vless://00000000-0000-0000-0000-000000000002@example.com:443?encryption=none&security=reality&type=tcp#n1",
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}
	sid := sub.ID
	nid := node.ID
	row := WgInterfaceRow{
		Name:           "t1",
		PrivateKeyEnc:  "enc-placeholder-32bytes-min______",
		PublicKey:      "pubkey________________________________________=",
		ListenPort:     51821,
		WgAddress:      "10.100.2.1/24",
		SubscriptionID: &sid,
		ActiveNodeID:   &nid,
		Status:         string(domain.WgStatusStopped),
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewTunnelRepo(db)
	if err := repo.ClearActiveNodeID(ctx, row.ID); err != nil {
		t.Fatal(err)
	}
	var after WgInterfaceRow
	if err := db.First(&after, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if after.ActiveNodeID != nil {
		t.Fatalf("expected active_node_id NULL, got %#v", after.ActiveNodeID)
	}
}

// TestTunnelRepo_Delete_cascadesTunnelNodes is a regression for the FK-constraint failure
// observed when deleting a tunnel that had any node assigned via the tunnel_nodes junction
// table. The DB connection enables foreign_keys=1 explicitly so the test reproduces the
// production failure mode (the other tests in this file run with FKs off by default).
func TestTunnelRepo_Delete_cascadesTunnelNodes(t *testing.T) {
	ctx := context.Background()
	dsn := "file:" + filepath.Join(t.TempDir(), "tunfk.db") + "?_pragma=foreign_keys(1)"
	db, err := gorm.Open(gormsqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		x, err := db.DB()
		if err == nil {
			_ = x.Close()
		}
	})
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}

	sub := SubscriptionRow{Name: "__manual__", URL: "", RefreshInterval: 86400, Status: string(domain.SubStatusInactive)}
	if err := db.Create(&sub).Error; err != nil {
		t.Fatal(err)
	}
	node := VlessNodeRow{
		SubscriptionID: sub.ID,
		DisplayName:    "n1",
		UUID:           "00000000-0000-0000-0000-000000000003",
		Address:        "example.com",
		Port:           443,
		RawURI:         "vless://00000000-0000-0000-0000-000000000003@example.com:443?encryption=none&security=reality&type=tcp#n1",
	}
	if err := db.Create(&node).Error; err != nil {
		t.Fatal(err)
	}

	// Two tunnels: the doomed one plus a survivor we use to assert isolation of the delete.
	doomed := WgInterfaceRow{
		Name:          "doomed",
		PrivateKeyEnc: "enc-placeholder-32bytes-min______",
		PublicKey:     "pubkey_doomed_________________________________=",
		ListenPort:    51840,
		WgAddress:     "10.100.40.1/24",
		Status:        string(domain.WgStatusStopped),
	}
	survivor := WgInterfaceRow{
		Name:          "survivor",
		PrivateKeyEnc: "enc-placeholder-32bytes-min______",
		PublicKey:     "pubkey_survivor_______________________________=",
		ListenPort:    51841,
		WgAddress:     "10.100.41.1/24",
		Status:        string(domain.WgStatusStopped),
	}
	if err := db.Create(&doomed).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&survivor).Error; err != nil {
		t.Fatal(err)
	}

	// Junction row on each tunnel — the survivor's row must survive the delete, the
	// doomed's row is what currently blocks the DELETE on wg_interfaces.
	if err := db.Create(&TunnelNodeRow{InterfaceID: doomed.ID, NodeID: node.ID, Position: 0}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&TunnelNodeRow{InterfaceID: survivor.ID, NodeID: node.ID, Position: 0}).Error; err != nil {
		t.Fatal(err)
	}

	// A peer and matching stats rows so we exercise the full cleanup path.
	peer := WgPeerRow{
		InterfaceID:   doomed.ID,
		Name:          "p1",
		PublicKey:     "peerpub_doomed________________________________=",
		ClientAddress: "10.100.40.2/32",
	}
	if err := db.Create(&peer).Error; err != nil {
		t.Fatal(err)
	}
	ifaceID := doomed.ID
	peerID := peer.ID
	if err := db.Create(&StatsSnapshotRow{InterfaceID: &ifaceID, RxBytes: 1, TxBytes: 2, RxRate: 0, TxRate: 0}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&StatsSnapshotRow{PeerID: &peerID, RxBytes: 1, TxBytes: 2, RxRate: 0, TxRate: 0}).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewTunnelRepo(db)
	if err := repo.Delete(ctx, doomed.ID); err != nil {
		t.Fatalf("Delete returned %v (FK enforcement on)", err)
	}

	var doomedRow WgInterfaceRow
	if err := db.First(&doomedRow, doomed.ID).Error; err == nil {
		t.Fatalf("doomed interface still present: %+v", doomedRow)
	}
	var doomedJunctions int64
	if err := db.Model(&TunnelNodeRow{}).Where("interface_id = ?", doomed.ID).Count(&doomedJunctions).Error; err != nil {
		t.Fatal(err)
	}
	if doomedJunctions != 0 {
		t.Fatalf("tunnel_nodes for doomed interface remain: %d", doomedJunctions)
	}
	var survivorJunctions int64
	if err := db.Model(&TunnelNodeRow{}).Where("interface_id = ?", survivor.ID).Count(&survivorJunctions).Error; err != nil {
		t.Fatal(err)
	}
	if survivorJunctions != 1 {
		t.Fatalf("survivor's tunnel_nodes were touched: count=%d, want 1", survivorJunctions)
	}
	var peerCount int64
	if err := db.Model(&WgPeerRow{}).Where("interface_id = ?", doomed.ID).Count(&peerCount).Error; err != nil {
		t.Fatal(err)
	}
	if peerCount != 0 {
		t.Fatalf("doomed peers remain: %d", peerCount)
	}
}

func TestTunnelRepo_ListRunningIDsTracksPersistedStatus(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(gormsqlite.Open("file:"+filepath.Join(t.TempDir(), "tunrunning.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		x, err := db.DB()
		if err == nil {
			_ = x.Close()
		}
	})
	if err := AutoMigrate(db); err != nil {
		t.Fatal(err)
	}

	running := WgInterfaceRow{
		Name:          "running",
		PrivateKeyEnc: "enc-placeholder-32bytes-min______",
		PublicKey:     "pubkey_running________________________________=",
		ListenPort:    51830,
		WgAddress:     "10.100.30.1/24",
		Status:        string(domain.WgStatusRunning),
	}
	stopped := WgInterfaceRow{
		Name:          "stopped",
		PrivateKeyEnc: "enc-placeholder-32bytes-min______",
		PublicKey:     "pubkey_stopped________________________________=",
		ListenPort:    51831,
		WgAddress:     "10.100.31.1/24",
		Status:        string(domain.WgStatusStopped),
	}
	if err := db.Create(&running).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&stopped).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewTunnelRepo(db)
	ids, err := repo.ListRunningIDs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != running.ID {
		t.Fatalf("running ids = %#v, want [%d]", ids, running.ID)
	}

	if err := repo.UpdateStatus(ctx, running.ID, domain.WgStatusStopped, ""); err != nil {
		t.Fatal(err)
	}
	ids, err = repo.ListRunningIDs(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 {
		t.Fatalf("running ids after explicit stop = %#v, want []", ids)
	}
}

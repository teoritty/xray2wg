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

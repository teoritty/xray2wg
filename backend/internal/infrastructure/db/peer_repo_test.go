package sqldb

import (
	"context"
	"path/filepath"
	"testing"

	"xray2wg/backend/internal/domain"

	gormsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestPeerRepo_UpdateTraffic_accumulatesAcrossDeviceRestart(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(gormsqlite.Open("file:"+filepath.Join(t.TempDir(), "traffic.db")), &gorm.Config{})
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
	iface := WgInterfaceRow{
		Name: "alpha", PrivateKeyEnc: "enc-placeholder-32bytes-min______",
		PublicKey: "pubA____________________________________________=",
		ListenPort: 51820, WgAddress: "10.100.1.1/24",
		Status: string(domain.WgStatusStopped),
	}
	if err := db.Create(&iface).Error; err != nil {
		t.Fatal(err)
	}
	peer := WgPeerRow{InterfaceID: iface.ID, Name: "c1", PublicKey: "pk_______________________________________________=", ClientAddress: "10.100.1.2/32"}
	if err := db.Create(&peer).Error; err != nil {
		t.Fatal(err)
	}
	repo := NewPeerRepo(db)

	// First poll: raw 1000/2000 → delta = 1000/2000 (last_seen was 0).
	rx, tx, err := repo.UpdateTraffic(ctx, peer.ID, nil, 1000, 2000)
	if err != nil {
		t.Fatal(err)
	}
	if rx != 1000 || tx != 2000 {
		t.Fatalf("first poll accum: rx=%d tx=%d (want 1000/2000)", rx, tx)
	}

	// Second poll: raw advanced 1500/3000 → delta = 500/1000, accum = 1500/3000.
	rx, tx, err = repo.UpdateTraffic(ctx, peer.ID, nil, 1500, 3000)
	if err != nil {
		t.Fatal(err)
	}
	if rx != 1500 || tx != 3000 {
		t.Fatalf("second poll accum: rx=%d tx=%d (want 1500/3000)", rx, tx)
	}

	// Device restart: raw drops back to 200/400. Treat the new value as fresh-from-zero,
	// add it whole. accum should become 1700/3400, NOT collapse to 200/400.
	rx, tx, err = repo.UpdateTraffic(ctx, peer.ID, nil, 200, 400)
	if err != nil {
		t.Fatal(err)
	}
	if rx != 1700 || tx != 3400 {
		t.Fatalf("post-restart accum: rx=%d tx=%d (want 1700/3400 — regression #5)", rx, tx)
	}

	// Next poll after restart: raw 500/900 → delta = 300/500, accum = 2000/3900.
	rx, tx, err = repo.UpdateTraffic(ctx, peer.ID, nil, 500, 900)
	if err != nil {
		t.Fatal(err)
	}
	if rx != 2000 || tx != 3900 {
		t.Fatalf("post-restart second poll: rx=%d tx=%d (want 2000/3900)", rx, tx)
	}
}

func TestPeerRepo_UpdateTraffic_missingPeerReturnsNotFound(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(gormsqlite.Open("file:"+filepath.Join(t.TempDir(), "missing.db")), &gorm.Config{})
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
	repo := NewPeerRepo(db)
	if _, _, err := repo.UpdateTraffic(ctx, 999, nil, 100, 200); err != domain.ErrNotFound {
		t.Fatalf("missing peer: want ErrNotFound, got %v", err)
	}
}

func TestPeerRepo_ListAllWithTunnel_joinsTunnelName(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(gormsqlite.Open("file:"+filepath.Join(t.TempDir(), "peer.db")), &gorm.Config{})
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

	ifaceA := WgInterfaceRow{
		Name:          "alpha",
		PrivateKeyEnc: "enc-placeholder-32bytes-min______",
		PublicKey:     "pubA____________________________________________=",
		ListenPort:    51820,
		WgAddress:     "10.100.1.1/24",
		Status:        string(domain.WgStatusStopped),
	}
	ifaceB := WgInterfaceRow{
		Name:          "beta",
		PrivateKeyEnc: "enc-placeholder-32bytes-min______",
		PublicKey:     "pubB____________________________________________=",
		ListenPort:    51821,
		WgAddress:     "10.100.2.1/24",
		Status:        string(domain.WgStatusStopped),
	}
	if err := db.Create(&ifaceA).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&ifaceB).Error; err != nil {
		t.Fatal(err)
	}

	p1 := WgPeerRow{
		InterfaceID:   ifaceA.ID,
		Name:          "client-one",
		PublicKey:     "peerpub_________________________________________=",
		ClientAddress: "10.100.1.2/32",
	}
	p2 := WgPeerRow{
		InterfaceID:   ifaceB.ID,
		Name:          "client-two",
		PublicKey:     "peerpub2________________________________________=",
		ClientAddress: "10.100.2.2/32",
	}
	if err := db.Create(&p1).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&p2).Error; err != nil {
		t.Fatal(err)
	}

	repo := NewPeerRepo(db)
	list, err := repo.ListAllWithTunnel(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("len %d", len(list))
	}
	byName := map[string]*domain.PeerWithTunnel{}
	for _, it := range list {
		byName[it.Name] = it
	}
	if byName["client-one"].TunnelName != "alpha" || byName["client-one"].InterfaceID != ifaceA.ID {
		t.Fatalf("client-one: %+v", byName["client-one"])
	}
	if byName["client-two"].TunnelName != "beta" || byName["client-two"].InterfaceID != ifaceB.ID {
		t.Fatalf("client-two: %+v", byName["client-two"])
	}
}

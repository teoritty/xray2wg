package sqldb

import (
	"context"
	"path/filepath"
	"testing"

	"xray2wg/backend/internal/domain"

	gormsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

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

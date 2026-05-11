package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"xray2wg/backend/internal/domain"
	sqldb "xray2wg/backend/internal/infrastructure/db"

	gormsqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const sampleVless = `vless://550e8400-e29b-41d4-a716-446655440000@vpn.example.com:443?encryption=none&security=reality&sni=vpn.example.com&fp=chrome&pbk=AbCdEf123456&type=tcp&flow=xtls-rprx-vision&sid=a1b2c3d4#MyNode`

const otherVless = `vless://660e8400-e29b-41d4-a716-446655440001@other.example.com:443?encryption=none&security=reality&sni=other.example.com&fp=chrome&pbk=AbCdEf123456&type=tcp&flow=xtls-rprx-vision&sid=a1b2c3d4#Other`

func openSubscriptionTestDB(t *testing.T, file string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(gormsqlite.Open("file:"+filepath.Join(t.TempDir(), file)), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		x, err := db.DB()
		if err == nil {
			_ = x.Close()
		}
	})
	return db
}

func TestSubscriptionService_AddManualVlessNode_insertsAndCounts(t *testing.T) {
	ctx := context.Background()
	db := openSubscriptionTestDB(t, "submanual1.db")
	if err := sqldb.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := sqldb.NewSubscriptionRepo(db)
	svc := NewSubscriptionService(repo, NewEventLog(8))

	n, err := svc.AddManualVlessNode(ctx, sampleVless)
	if err != nil {
		t.Fatal(err)
	}
	if n.ID == 0 || n.SubscriptionID == 0 {
		t.Fatalf("expected non-zero ids: %#v", n)
	}
	sub, err := repo.GetByID(ctx, n.SubscriptionID)
	if err != nil {
		t.Fatal(err)
	}
	if sub.Name != ManualSubscriptionName {
		t.Fatalf("name: %q", sub.Name)
	}
	if sub.NodeCount != 1 {
		t.Fatalf("node_count: %d", sub.NodeCount)
	}
	if sub.Status != domain.SubStatusActive {
		t.Fatalf("status: %s", sub.Status)
	}
}

func TestSubscriptionService_AddManualVlessNode_duplicateURI(t *testing.T) {
	ctx := context.Background()
	db := openSubscriptionTestDB(t, "submanual2.db")
	if err := sqldb.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := sqldb.NewSubscriptionRepo(db)
	svc := NewSubscriptionService(repo, NewEventLog(8))

	first, err := svc.AddManualVlessNode(ctx, sampleVless)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.AddManualVlessNode(ctx, sampleVless)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	nodes, err := repo.ListNodes(ctx, first.SubscriptionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("len(nodes)=%d", len(nodes))
	}
}

func TestSubscriptionService_UpdateManualVlessNode_changesFields(t *testing.T) {
	ctx := context.Background()
	db := openSubscriptionTestDB(t, "submanualupd.db")
	if err := sqldb.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := sqldb.NewSubscriptionRepo(db)
	svc := NewSubscriptionService(repo, NewEventLog(8))

	n, err := svc.AddManualVlessNode(ctx, sampleVless)
	if err != nil {
		t.Fatal(err)
	}
	updatedURI := `vless://550e8400-e29b-41d4-a716-446655440000@newhost.example.net:8443?encryption=none&security=reality&sni=newhost.example.net&fp=chrome&pbk=AbCdEf123456&type=tcp&flow=xtls-rprx-vision&sid=a1b2c3d4#Renamed`
	out, err := svc.UpdateManualVlessNode(ctx, n.ID, updatedURI)
	if err != nil {
		t.Fatal(err)
	}
	if out.Address != "newhost.example.net" || out.Port != 8443 {
		t.Fatalf("unexpected node: %#v", out)
	}
	if out.RawURI != updatedURI {
		t.Fatalf("raw_uri: %q", out.RawURI)
	}
}

func TestSubscriptionService_UpdateManualVlessNode_duplicateURI(t *testing.T) {
	ctx := context.Background()
	db := openSubscriptionTestDB(t, "submanualupddup.db")
	if err := sqldb.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := sqldb.NewSubscriptionRepo(db)
	svc := NewSubscriptionService(repo, NewEventLog(8))

	a, err := svc.AddManualVlessNode(ctx, sampleVless)
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.AddManualVlessNode(ctx, otherVless)
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.UpdateManualVlessNode(ctx, b.ID, sampleVless)
	if !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("want ErrConflict, got %v", err)
	}
	firstAgain, err := repo.GetNode(ctx, a.ID)
	if err != nil {
		t.Fatal(err)
	}
	if firstAgain.RawURI != sampleVless {
		t.Fatalf("first node raw_uri changed: %q", firstAgain.RawURI)
	}
}

func TestSubscriptionService_DeleteManualVlessNode_ok(t *testing.T) {
	ctx := context.Background()
	db := openSubscriptionTestDB(t, "submanualdel.db")
	if err := sqldb.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := sqldb.NewSubscriptionRepo(db)
	svc := NewSubscriptionService(repo, NewEventLog(8))

	n, err := svc.AddManualVlessNode(ctx, sampleVless)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteManualVlessNode(ctx, n.ID); err != nil {
		t.Fatal(err)
	}
	nodes, err := repo.ListNodes(ctx, n.SubscriptionID)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 0 {
		t.Fatalf("len(nodes)=%d", len(nodes))
	}
	sub, err := repo.GetByID(ctx, n.SubscriptionID)
	if err != nil {
		t.Fatal(err)
	}
	if sub.NodeCount != 0 {
		t.Fatalf("node_count: %d", sub.NodeCount)
	}
}

func TestSubscriptionService_DeleteManualVlessNode_blockedByTunnel(t *testing.T) {
	ctx := context.Background()
	db := openSubscriptionTestDB(t, "submanualdelblk.db")
	if err := sqldb.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := sqldb.NewSubscriptionRepo(db)
	svc := NewSubscriptionService(repo, NewEventLog(8))

	n, err := svc.AddManualVlessNode(ctx, sampleVless)
	if err != nil {
		t.Fatal(err)
	}
	sid := n.SubscriptionID
	nid := n.ID
	row := sqldb.WgInterfaceRow{
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

	err = svc.DeleteManualVlessNode(ctx, n.ID)
	if !errors.Is(err, domain.ErrNodeInUse) {
		t.Fatalf("want ErrNodeInUse, got %v", err)
	}
}

func TestSubscriptionService_DeleteManualVlessNode_nonManualNode(t *testing.T) {
	ctx := context.Background()
	db := openSubscriptionTestDB(t, "submanualdelnm.db")
	if err := sqldb.AutoMigrate(db); err != nil {
		t.Fatal(err)
	}
	repo := sqldb.NewSubscriptionRepo(db)
	svc := NewSubscriptionService(repo, NewEventLog(8))

	sub := &domain.Subscription{Name: "feed", URL: "http://example.com/sub", RefreshInterval: 3600}
	if err := repo.Create(ctx, sub); err != nil {
		t.Fatal(err)
	}
	node := &domain.VlessNode{
		SubscriptionID: sub.ID,
		DisplayName:    "x",
		UUID:           "770e8400-e29b-41d4-a716-446655440002",
		Address:        "x.example.com",
		Port:           443,
		RawURI:         "vless://770e8400-e29b-41d4-a716-446655440002@x.example.com:443?encryption=none&security=reality&type=tcp#x",
	}
	if err := repo.InsertNodes(ctx, []*domain.VlessNode{node}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteManualVlessNode(ctx, node.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/installer"
	adminstore "github.com/dujiao-next/internal/modules/identity/admin/infrastructure/gormstore"
	settingsstore "github.com/dujiao-next/internal/modules/settings/infrastructure/gormstore"
	"github.com/dujiao-next/internal/platform/database/gormdb"
	"golang.org/x/crypto/bcrypt"
)

func TestWebInstallationCreatesClosedDatabaseWithAdministratorAndSite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dujiao.db")
	input := installer.Input{AdminUser: "store-owner", AdminPassword: "StrongPassword99", SiteURL: "https://shop.example.com"}
	if err := initializeWebInstallation(context.Background(), path, input); err != nil {
		t.Fatal(err)
	}
	previous := gormdb.DB
	if err := gormdb.InitDB("sqlite", path, gormdb.DBPoolConfig{MaxOpenConns: 1, MaxIdleConns: 1}, "release"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db, _ := gormdb.DB.DB(); _ = db.Close(); gormdb.DB = previous })
	admin, err := adminstore.New(gormdb.DB).GetByUsername(input.AdminUser)
	if err != nil || admin == nil || !admin.IsSuper {
		t.Fatal("administrator not initialized", err)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(input.AdminPassword)); err != nil {
		t.Fatal(err)
	}
	value, found, err := settingsstore.New(gormdb.DB).GetByKey(constants.SettingKeySiteConfig)
	if err != nil || !found || value["currency"] != "CNY" || value["brand"].(map[string]interface{})["site_url"] != input.SiteURL {
		t.Fatal("website configuration missing", value, err)
	}
}

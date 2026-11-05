package main

import (
	"context"

	databasemigrations "github.com/dujiao-next/internal/bootstrap/database/migrations"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/installer"
	adminapplication "github.com/dujiao-next/internal/modules/identity/admin/application"
	adminstore "github.com/dujiao-next/internal/modules/identity/admin/infrastructure/gormstore"
	settingsstore "github.com/dujiao-next/internal/modules/settings/infrastructure/gormstore"
	"github.com/dujiao-next/internal/platform/database/gormdb"
	"github.com/dujiao-next/internal/shared/jsonmap"
)

// The normal migration and administrator bootstrap run against an unpublished fresh database.
func initializeWebInstallation(ctx context.Context, databasePath string, input installer.Input) (result error) {
	previous := gormdb.DB
	defer func() { gormdb.DB = previous }()
	if err := gormdb.InitDB("sqlite", databasePath, gormdb.DBPoolConfig{MaxOpenConns: 1, MaxIdleConns: 1}, "release"); err != nil {
		return err
	}
	db, err := gormdb.DB.DB()
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); result == nil {
			result = err
		}
	}()
	gormdb.DB = gormdb.DB.WithContext(ctx)
	if err := databasemigrations.AutoMigrate(); err != nil {
		return err
	}
	if _, err := adminapplication.InitDefaultAdmin(adminstore.New(gormdb.DB), input.AdminUser, input.AdminPassword); err != nil {
		return err
	}
	_, err = settingsstore.New(gormdb.DB).Upsert(constants.SettingKeySiteConfig, jsonmap.JSON{
		constants.SettingFieldSiteCurrency: "CNY",
		"brand":                            map[string]interface{}{"site_name": "Dujiao Next", "site_url": input.SiteURL},
	})
	if err != nil {
		return err
	}
	return gormdb.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)").Error
}

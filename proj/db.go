package proj

import (
	"context"

	"github.com/Lyndon-Zhang/gira"
	"github.com/Lyndon-Zhang/gira/db"
)

func UseAccountDbClient(ctx context.Context, driver gira.DbDao) error {
	if client, err := db.NewConfigDbClient(ctx, gira.ACCOUNTDB_NAME, *Config.Db[gira.ACCOUNTDB_NAME]); err != nil {
		return err
	} else if err := driver.UseClient(client); err != nil {
		return err
	} else {
		return nil
	}
}

func UseGameDbClient(ctx context.Context, driver gira.DbDao) error {
	if client, err := db.NewConfigDbClient(ctx, gira.GAMEDB_NAME, *Config.Db[gira.GAMEDB_NAME]); err != nil {
		return err
	} else if err := driver.UseClient(client); err != nil {
		return err
	} else {
		return nil
	}
}

func UseResourceDbClient(ctx context.Context, driver gira.DbDao) error {
	if client, err := db.NewConfigDbClient(ctx, gira.RESOURCEDB_NAME, *Config.Db[gira.RESOURCEDB_NAME]); err != nil {
		return err
	} else if err := driver.UseClient(client); err != nil {
		return err
	} else {
		return nil
	}
}

func NewResourceDbClient(ctx context.Context) (gira.DbClient, error) {
	if client, err := db.NewConfigDbClient(ctx, gira.RESOURCEDB_NAME, *Config.Db[gira.RESOURCEDB_NAME]); err != nil {
		return nil, err
	} else {
		return client, nil
	}
}

func NewAdminCacheClient(ctx context.Context) (gira.DbClient, error) {
	if client, err := db.NewConfigDbClient(ctx, gira.ADMINCACHE_NAME, *Config.Db[gira.ADMINCACHE_NAME]); err != nil {
		return nil, err
	} else {
		return client, nil
	}
}

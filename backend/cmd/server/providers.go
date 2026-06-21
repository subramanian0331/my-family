package main

import (
	googleclient "github.com/subbu/family_tree/client/google"
	postgresclient "github.com/subbu/family_tree/client/postgres"
	storageclient "github.com/subbu/family_tree/client/storage"
	"github.com/subbu/family_tree/config"
	authservice "github.com/subbu/family_tree/services/auth"
	userservice "github.com/subbu/family_tree/services/user"
)

func providePostgresClient(cfg config.Config) (postgresclient.Client, error) {
	return postgresclient.New(cfg.DatabaseURL)
}

func provideStorageClient(cfg config.Config) (storageclient.Client, error) {
	return storageclient.New(cfg.UploadDir)
}

func provideAuthService(
	google googleclient.Client,
	db postgresclient.Client,
	users userservice.Service,
	cfg config.Config,
) authservice.Service {
	return authservice.NewService(google, db, users, cfg.JWTSecret, cfg.SiteAdminEmail)
}
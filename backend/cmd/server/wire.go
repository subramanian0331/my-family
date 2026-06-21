//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/subbu/family_tree/client/google"
	"github.com/subbu/family_tree/config"
	adminhandler "github.com/subbu/family_tree/handlers/admin"
	authhandler "github.com/subbu/family_tree/handlers/auth"
	familyhandler "github.com/subbu/family_tree/handlers/family"
	gedcomhandler "github.com/subbu/family_tree/handlers/gedcom"
	"github.com/subbu/family_tree/handlers"
	"github.com/subbu/family_tree/handlers/health"
	invitehandler "github.com/subbu/family_tree/handlers/invite"
	personhandler "github.com/subbu/family_tree/handlers/person"
	photohandler "github.com/subbu/family_tree/handlers/photo"
	relationshiphandler "github.com/subbu/family_tree/handlers/relationship"
	searchhandler "github.com/subbu/family_tree/handlers/search"
	treehandler "github.com/subbu/family_tree/handlers/tree"
	"github.com/subbu/family_tree/server"
	familyservice "github.com/subbu/family_tree/services/family"
	gedcomservice "github.com/subbu/family_tree/services/gedcom"
	inviteservice "github.com/subbu/family_tree/services/invite"
	personservice "github.com/subbu/family_tree/services/person"
	photoservice "github.com/subbu/family_tree/services/photo"
	relationshipservice "github.com/subbu/family_tree/services/relationship"
	searchservice "github.com/subbu/family_tree/services/search"
	userservice "github.com/subbu/family_tree/services/user"
)

func InitializeServer() (*server.Server, error) {
	wire.Build(
		config.Load,
		provideGoogleClient,
		providePostgresClient,
		provideStorageClient,
		provideEmailClient,
		userservice.NewService,
		provideAuthService,
		familyservice.NewService,
		inviteservice.NewService,
		personservice.NewService,
		relationshipservice.NewService,
		photoservice.NewService,
		searchservice.NewService,
		gedcomservice.NewService,
		authhandler.NewHandler,
		familyhandler.NewHandler,
		invitehandler.NewHandler,
		personhandler.NewHandler,
		relationshiphandler.NewHandler,
		searchhandler.NewHandler,
		photohandler.NewHandler,
		treehandler.NewHandler,
		gedcomhandler.NewHandler,
		adminhandler.NewHandler,
		health.NewHandler,
		wire.Struct(new(handlers.Dependencies), "*"),
		handlers.NewRouter,
		server.New,
	)
	return nil, nil
}

func provideGoogleClient(cfg config.Config) google.Client {
	redirectURL := cfg.FrontendURL + "/api/auth/google/callback"
	return google.New(cfg.GoogleClientID, cfg.GoogleClientSecret, redirectURL)
}
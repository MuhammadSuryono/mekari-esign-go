package main

import (
	"go.uber.org/fx"

	"mekari-esign/internal/config"
	deliveryhttp "mekari-esign/internal/delivery/http"
	"mekari-esign/internal/infrastructure/database"
	"mekari-esign/internal/infrastructure/document"
	"mekari-esign/internal/infrastructure/httpclient"
	"mekari-esign/internal/infrastructure/logger"
	"mekari-esign/internal/infrastructure/nav"
	"mekari-esign/internal/infrastructure/oauth2"
	"mekari-esign/internal/infrastructure/redis"
	"mekari-esign/internal/infrastructure/repository"
	"mekari-esign/internal/server"
	"mekari-esign/internal/usecase"
)

func main() {
	fx.New(
		// Configuration
		config.Module,

		// Infrastructure
		logger.Module,
		database.Module,
		redis.Module,
		oauth2.Module,
		document.Module,
		httpclient.Module,
		nav.Module,
		repository.Module,

		// Business Logic
		usecase.Module,

		// Delivery
		deliveryhttp.Module,

		// Server
		server.Module,
	).Run()
}
